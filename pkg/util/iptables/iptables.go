package iptables

import (
	"bytes"
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/go-kit/log"
	"golang.org/x/sys/unix"
	utilerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/apimachinery/pkg/util/wait"
	utilexec "k8s.io/utils/exec"

	utilversion "github.com/sapcc/kubernikus/pkg/util/version"
)

type RulePosition string

const (
	Prepend RulePosition = "-I"
	Append  RulePosition = "-A"
)

// An injectable interface for running iptables commands.  Implementations must be goroutine-safe.
type Interface interface {
	// GetVersion returns the "X.Y.Z" version string for iptables.
	GetVersion() (string, error)
	// EnsureChain checks if the specified chain exists and, if not, creates it.  If the chain existed, return true.
	EnsureChain(table Table, chain Chain) (bool, error)
	// FlushChain clears the specified chain.  If the chain did not exist, return error.
	FlushChain(table Table, chain Chain) error
	// DeleteChain deletes the specified chain.  If the chain did not exist, return error.
	DeleteChain(table Table, chain Chain) error
	// EnsureRule checks if the specified rule is present and, if not, creates it.  If the rule existed, return true.
	EnsureRule(position RulePosition, table Table, chain Chain, args ...string) (bool, error)
	// DeleteRule checks if the specified rule is present and, if so, deletes it.
	DeleteRule(table Table, chain Chain, args ...string) error
	// IsIpv6 returns true if this is managing ipv6 tables
	IsIpv6() bool
	// SaveInto calls `iptables-save` for table and stores result in a given buffer.
	SaveInto(table Table, buffer *bytes.Buffer) error
	// Restore runs `iptables-restore` passing data through []byte.
	// table is the Table to restore
	// data should be formatted like the output of SaveInto()
	// flush sets the presence of the "--noflush" flag. see: FlushFlag
	// counters sets the "--counters" flag. see: RestoreCountersFlag
	Restore(table Table, data []byte, flush FlushFlag, counters RestoreCountersFlag) error
	// RestoreAll is the same as Restore except that no table is specified.
	RestoreAll(data []byte, flush FlushFlag, counters RestoreCountersFlag) error
	// AddReloadFunc adds a function to call on iptables reload
	AddReloadFunc(reloadFunc func())
	// Destroy cleans up resources used by the Interface
	Destroy()
}

type Protocol byte

const (
	ProtocolIpv4 Protocol = iota + 1
	ProtocolIpv6
)

type Table string

const (
	TableNAT    Table = "nat"
	TableFilter Table = "filter"
)

type Chain string

const (
	ChainPostrouting Chain = "POSTROUTING"
	ChainPrerouting  Chain = "PREROUTING"
	ChainOutput      Chain = "OUTPUT"
	ChainInput       Chain = "INPUT"
)

const (
	cmdIPTablesSave    string = "iptables-save"
	cmdIPTablesRestore string = "iptables-restore"
	cmdIPTables        string = "iptables"
	cmdIp6tables       string = "ip6tables"
)

// Option flag for Restore
type RestoreCountersFlag bool

const RestoreCounters RestoreCountersFlag = true
const NoRestoreCounters RestoreCountersFlag = false

// Option flag for Flush
type FlushFlag bool

const FlushTables FlushFlag = true
const NoFlushTables FlushFlag = false

// Versions of iptables less than this do not support the -C / --check flag
// (test whether a rule exists).
const MinCheckVersion = "1.4.11"

// Minimum iptables versions supporting the -w and -w2 flags
const MinWaitVersion = "1.4.20"
const MinWait2Version = "1.4.22"

const LockfilePath16x = "/run/xtables.lock"

// runner implements Interface in terms of exec("iptables").
type runner struct {
	mu              sync.Mutex
	exec            utilexec.Interface
	protocol        Protocol
	hasCheck        bool
	waitFlag        []string
	restoreWaitFlag []string
	lockfilePath    string
	logger          log.Logger

	reloadFuncs []func()
}

// newInternal returns a new Interface which will exec iptables, and allows the
// caller to change the iptables-restore lockfile path
func newInternal(exec utilexec.Interface, protocol Protocol, lockfilePath string, logger log.Logger) Interface {
	vstring, err := getIPTablesVersionString(exec)
	if err != nil {
		logger.Log(
			"msg", "Error checking iptables version",
			"min_version", MinCheckVersion,
			"err", err)
		vstring = MinCheckVersion
	}

	if lockfilePath == "" {
		lockfilePath = LockfilePath16x
	}

	runner := &runner{
		exec:            exec,
		protocol:        protocol,
		hasCheck:        getIPTablesHasCheckCommand(vstring, logger),
		waitFlag:        getIPTablesWaitFlag(vstring, logger),
		restoreWaitFlag: getIPTablesRestoreWaitFlag(exec, logger),
		lockfilePath:    lockfilePath,
		logger:          logger,
	}

	return runner
}

// New returns a new Interface which will exec iptables.
func New(exec utilexec.Interface, protocol Protocol, logger log.Logger) Interface {
	return newInternal(exec, protocol, "", logger)
}

// Destroy is part of Interface.
func (runner *runner) Destroy() {
}

// GetVersion returns the version string.
func (runner *runner) GetVersion() (string, error) {
	return getIPTablesVersionString(runner.exec)
}

// EnsureChain is part of Interface.
func (runner *runner) EnsureChain(table Table, chain Chain) (bool, error) {
	fullArgs := makeFullArgs(table, chain)

	runner.mu.Lock()
	defer runner.mu.Unlock()

	out, err := runner.run(opCreateChain, fullArgs)
	if err != nil {
		if ee, ok := err.(utilexec.ExitError); ok {
			if ee.Exited() && ee.ExitStatus() == 1 {
				return true, nil
			}
		}
		return false, fmt.Errorf("error creating chain %q: %v: %s", chain, err, out)
	}
	return false, nil
}

// FlushChain is part of Interface.
func (runner *runner) FlushChain(table Table, chain Chain) error {
	fullArgs := makeFullArgs(table, chain)

	runner.mu.Lock()
	defer runner.mu.Unlock()

	out, err := runner.run(opFlushChain, fullArgs)
	if err != nil {
		return fmt.Errorf("error flushing chain %q: %v: %s", chain, err, out)
	}
	return nil
}

// DeleteChain is part of Interface.
func (runner *runner) DeleteChain(table Table, chain Chain) error {
	fullArgs := makeFullArgs(table, chain)

	runner.mu.Lock()
	defer runner.mu.Unlock()

	// TODO: we could call iptables -S first, ignore the output and check for non-zero return (more like DeleteRule)
	out, err := runner.run(opDeleteChain, fullArgs)
	if err != nil {
		return fmt.Errorf("error deleting chain %q: %v: %s", chain, err, out)
	}
	return nil
}

// EnsureRule is part of Interface.
func (runner *runner) EnsureRule(position RulePosition, table Table, chain Chain, args ...string) (bool, error) {
	fullArgs := makeFullArgs(table, chain, args...)

	runner.mu.Lock()
	defer runner.mu.Unlock()

	exists, err := runner.checkRule(table, chain, args...)
	if err != nil {
		return false, err
	}
	if exists {
		return true, nil
	}
	out, err := runner.run(operation(position), fullArgs)
	if err != nil {
		return false, fmt.Errorf("error appending rule: %v: %s", err, out)
	}
	return false, nil
}

// DeleteRule is part of Interface.
func (runner *runner) DeleteRule(table Table, chain Chain, args ...string) error {
	fullArgs := makeFullArgs(table, chain, args...)

	runner.mu.Lock()
	defer runner.mu.Unlock()

	exists, err := runner.checkRule(table, chain, args...)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}
	out, err := runner.run(opDeleteRule, fullArgs)
	if err != nil {
		return fmt.Errorf("error deleting rule: %v: %s", err, out)
	}
	return nil
}

func (runner *runner) IsIpv6() bool {
	return runner.protocol == ProtocolIpv6
}

// SaveInto is part of Interface.
func (runner *runner) SaveInto(table Table, buffer *bytes.Buffer) error {
	runner.mu.Lock()
	defer runner.mu.Unlock()

	// run and return
	args := []string{"-t", string(table)}
	runner.logger.Log(
		"msg", "running iptables-save",
		"args", args,
		"v", 4)
	cmd := runner.exec.Command(cmdIPTablesSave, args...)
	// Since CombinedOutput() doesn't support redirecting it to a buffer,
	// we need to workaround it by redirecting stdout and stderr to buffer
	// and explicitly calling Run() [CombinedOutput() underneath itself
	// creates a new buffer, redirects stdout and stderr to it and also
	// calls Run()].
	cmd.SetStdout(buffer)
	cmd.SetStderr(buffer)
	return cmd.Run()
}

// Restore is part of Interface.
func (runner *runner) Restore(table Table, data []byte, flush FlushFlag, counters RestoreCountersFlag) error {
	// setup args
	args := []string{"-T", string(table)}
	return runner.restoreInternal(args, data, flush, counters)
}

// RestoreAll is part of Interface.
func (runner *runner) RestoreAll(data []byte, flush FlushFlag, counters RestoreCountersFlag) error {
	// setup args
	args := make([]string, 0)
	return runner.restoreInternal(args, data, flush, counters)
}

type iptablesLocker interface {
	Close() error
}

// restoreInternal is the shared part of Restore/RestoreAll
func (runner *runner) restoreInternal(args []string, data []byte, flush FlushFlag, counters RestoreCountersFlag) error {
	runner.mu.Lock()
	defer runner.mu.Unlock()

	if !flush {
		args = append(args, "--noflush")
	}
	if counters {
		args = append(args, "--counters")
	}

	// Grab the iptables lock to prevent iptables-restore and iptables
	// from stepping on each other.  iptables-restore 1.6.2 will have
	// a --wait option like iptables itself, but that's not widely deployed.
	if len(runner.restoreWaitFlag) == 0 {
		locker, err := grabIptablesLocks(runner.lockfilePath)
		if err != nil {
			return err
		}
		defer func(locker iptablesLocker) {
			if err := locker.Close(); err != nil {
				runner.logger.Log(
					"msg", "Failed to close iptables locks",
					"err", err)
			}
		}(locker)
	}

	// run the command and return the output or an error including the output and error
	fullArgs := append(runner.restoreWaitFlag, args...)
	runner.logger.Log(
		"msg", "running iptables-restore",
		"args", fullArgs,
		"v", 4)
	cmd := runner.exec.Command(cmdIPTablesRestore, fullArgs...)
	cmd.SetStdin(bytes.NewBuffer(data))
	b, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%v (%s)", err, b)
	}
	return nil
}

func (runner *runner) iptablesCommand() string {
	if runner.IsIpv6() {
		return cmdIp6tables
	} else {
		return cmdIPTables
	}
}

func (runner *runner) run(op operation, args []string) ([]byte, error) {
	iptablesCmd := runner.iptablesCommand()

	fullArgs := append(runner.waitFlag, string(op))
	fullArgs = append(fullArgs, args...)
	runner.logger.Log(
		"msg", "running iptables",
		"op", string(op),
		"args", args,
		"v", 5)
	return runner.exec.Command(iptablesCmd, fullArgs...).CombinedOutput()
	// Don't log err here - callers might not think it is an error.
}

// Returns (bool, nil) if it was able to check the existence of the rule, or
// (<undefined>, error) if the process of checking failed.
func (runner *runner) checkRule(table Table, chain Chain, args ...string) (bool, error) {
	if runner.hasCheck {
		return runner.checkRuleUsingCheck(makeFullArgs(table, chain, args...))
	} else {
		return runner.checkRuleWithoutCheck(table, chain, args...)
	}
}

var hexnumRE = regexp.MustCompile("0x0+([0-9])")

func trimhex(s string) string {
	return hexnumRE.ReplaceAllString(s, "0x$1")
}

// Executes the rule check without using the "-C" flag, instead parsing iptables-save.
// Present for compatibility with <1.4.11 versions of iptables.  This is full
// of hack and half-measures.  We should nix this ASAP.
func (runner *runner) checkRuleWithoutCheck(table Table, chain Chain, args ...string) (bool, error) {
	runner.logger.Log(
		"msg", "running iptables-sav",
		"table", string(table))
	out, err := runner.exec.Command(cmdIPTablesSave, "-t", string(table)).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("error checking rule: %v", err)
	}

	// Sadly, iptables has inconsistent quoting rules for comments. Just remove all quotes.
	// Also, quoted multi-word comments (which are counted as a single arg)
	// will be unpacked into multiple args,
	// in order to compare against iptables-save output (which will be split at whitespace boundary)
	// e.g. a single arg('"this must be before the NodePort rules"') will be unquoted and unpacked into 7 args.
	var argsCopy []string
	for i := range args {
		tmpField := strings.Trim(args[i], "\"")
		tmpField = trimhex(tmpField)
		argsCopy = append(argsCopy, strings.Fields(tmpField)...)
	}
	argset := sets.NewString(argsCopy...)

	for _, line := range strings.Split(string(out), "\n") {
		var fields = strings.Fields(line)

		// Check that this is a rule for the correct chain, and that it has
		// the correct number of argument (+2 for "-A <chain name>")
		if !strings.HasPrefix(line, fmt.Sprintf("-A %s", string(chain))) || len(fields) != len(argsCopy)+2 {
			continue
		}

		// Sadly, iptables has inconsistent quoting rules for comments.
		// Just remove all quotes.
		for i := range fields {
			fields[i] = strings.Trim(fields[i], "\"")
			fields[i] = trimhex(fields[i])
		}

		// TODO: This misses reorderings e.g. "-x foo ! -y bar" will match "! -x foo -y bar"
		if sets.NewString(fields...).IsSuperset(argset) {
			return true, nil
		}
		runner.logger.Log(
			"msg", "DBG: fields is not a superset of args",
			"fields", fields,
			"args", args,
			"v", 5)
	}

	return false, nil
}

// Executes the rule check using the "-C" flag
func (runner *runner) checkRuleUsingCheck(args []string) (bool, error) {
	out, err := runner.run(opCheckRule, args)
	if err == nil {
		return true, nil
	}
	if ee, ok := err.(utilexec.ExitError); ok {
		// iptables uses exit(1) to indicate a failure of the operation,
		// as compared to a malformed commandline, for example.
		if ee.Exited() && ee.ExitStatus() == 1 {
			return false, nil
		}
	}
	return false, fmt.Errorf("error checking rule: %v: %s", err, out)
}

type operation string

const (
	opCreateChain operation = "-N"
	opFlushChain  operation = "-F"
	opDeleteChain operation = "-X"
	opAppendRule  operation = "-A" //nolint:deadcode,varcheck
	opCheckRule   operation = "-C"
	opDeleteRule  operation = "-D"
)

func makeFullArgs(table Table, chain Chain, args ...string) []string {
	return append([]string{string(chain), "-t", string(table)}, args...)
}

// Checks if iptables has the "-C" flag
func getIPTablesHasCheckCommand(vstring string, logger log.Logger) bool {
	minVersion, err := utilversion.ParseGeneric(MinCheckVersion)
	if err != nil {
		logger.Log(
			"msg", "MinCheckVersion is not a valid version string",
			"min_version", MinCheckVersion,
			"err", err)
		return true
	}
	version, err := utilversion.ParseGeneric(vstring)
	if err != nil {
		logger.Log(
			"msg", "vstring is not a valid version string",
			"vstring", vstring,
			"err", err)
		return true
	}
	return version.AtLeast(minVersion)
}

// Checks if iptables version has a "wait" flag
func getIPTablesWaitFlag(vstring string, logger log.Logger) []string {
	version, err := utilversion.ParseGeneric(vstring)
	if err != nil {
		logger.Log(
			"msg", "vstring is not a valid version string",
			"vstring", vstring,
			"err", err)
		return nil
	}

	minVersion, err := utilversion.ParseGeneric(MinWaitVersion)
	if err != nil {
		logger.Log(
			"msg", "MinWaitVersion is not a valid version string",
			"min_wait_version", MinWaitVersion,
			"err", err)
		return nil
	}
	if version.LessThan(minVersion) {
		return nil
	}

	minVersion, err = utilversion.ParseGeneric(MinWait2Version)
	if err != nil {
		logger.Log(
			"msg", "MinWait2Version is not a valid version string",
			"min_wait2_version", MinWait2Version,
			"err", err)
		return nil
	}
	if version.LessThan(minVersion) {
		return []string{"-w"}
	} else {
		return []string{"-w2"}
	}
}

// getIPTablesVersionString runs "iptables --version" to get the version string
// in the form "X.X.X"
func getIPTablesVersionString(exec utilexec.Interface) (string, error) {
	// this doesn't access mutable state so we don't need to use the interface / runner
	bytes, err := exec.Command(cmdIPTables, "--version").CombinedOutput()
	if err != nil {
		return "", err
	}
	versionMatcher := regexp.MustCompile("v([0-9]+(\\.[0-9]+)+)")
	match := versionMatcher.FindStringSubmatch(string(bytes))
	if match == nil {
		return "", fmt.Errorf("no iptables version found in string: %s", bytes)
	}
	return match[1], nil
}

// Checks if iptables-restore has a "wait" flag
// --wait support landed in v1.6.1+ right before --version support, so
// any version of iptables-restore that supports --version will also
// support --wait
func getIPTablesRestoreWaitFlag(exec utilexec.Interface, logger log.Logger) []string {
	vstring, err := getIPTablesRestoreVersionString(exec)
	if err != nil || vstring == "" {
		logger.Log(
			"msg", "couldn't get iptables-restore version; assuming it doesn't support --wait",
			"v", 3,
			"err", err)
		return nil
	}
	if _, err := utilversion.ParseGeneric(vstring); err != nil {
		logger.Log(
			"msg", "couldn't parse iptables-restore version; assuming it doesn't support --wait",
			"v", 3,
			"err", err)
		return nil
	}

	return []string{"--wait=2"}
}

// getIPTablesRestoreVersionString runs "iptables-restore --version" to get the version string
// in the form "X.X.X"
func getIPTablesRestoreVersionString(exec utilexec.Interface) (string, error) {
	// this doesn't access mutable state so we don't need to use the interface / runner

	// iptables-restore hasn't always had --version, and worse complains
	// about unrecognized commands but doesn't exit when it gets them.
	// Work around that by setting stdin to nothing so it exits immediately.
	cmd := exec.Command(cmdIPTablesRestore, "--version")
	cmd.SetStdin(bytes.NewReader([]byte{}))
	bytes, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	versionMatcher := regexp.MustCompile("v([0-9]+(\\.[0-9]+)+)")
	match := versionMatcher.FindStringSubmatch(string(bytes))
	if match == nil {
		return "", fmt.Errorf("no iptables version found in string: %s", bytes)
	}
	return match[1], nil
}

// AddReloadFunc is part of Interface
func (runner *runner) AddReloadFunc(reloadFunc func()) {
	runner.reloadFuncs = append(runner.reloadFuncs, reloadFunc)
}

// IsNotFoundError returns true if the error indicates "not found".  It parses
// the error string looking for known values, which is imperfect but works in
// practice.
func IsNotFoundError(err error) bool {
	es := err.Error()
	if strings.Contains(es, "No such file or directory") {
		return true
	}
	if strings.Contains(es, "No chain/target/match by that name") {
		return true
	}
	return false
}

type locker struct {
	lock16 *os.File
	lock14 *net.UnixListener
}

func (l *locker) Close() error {
	errList := []error{}
	if l.lock16 != nil {
		if err := l.lock16.Close(); err != nil {
			errList = append(errList, err)
		}
	}
	if l.lock14 != nil {
		if err := l.lock14.Close(); err != nil {
			errList = append(errList, err)
		}
	}
	return utilerrors.NewAggregate(errList)
}

func grabIptablesLocks(lockfilePath string) (iptablesLocker, error) {
	var err error
	var success bool

	l := &locker{}
	defer func(l *locker) {
		// Clean up immediately on failure
		if !success {
			l.Close()
		}
	}(l)

	// Grab both 1.6.x and 1.4.x-style locks; we don't know what the
	// iptables-restore version is if it doesn't support --wait, so we
	// can't assume which lock method it'll use.

	// Roughly duplicate iptables 1.6.x xtables_lock() function.
	l.lock16, err = os.OpenFile(lockfilePath, os.O_CREATE, 0600)
	if err != nil {
		return nil, fmt.Errorf("failed to open iptables lock %s: %v", lockfilePath, err)
	}

	if err := wait.PollImmediate(200*time.Millisecond, 2*time.Second, func() (bool, error) {
		if err := grabIptablesFileLock(l.lock16); err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to acquire new iptables lock: %v", err)
	}

	// Roughly duplicate iptables 1.4.x xtables_lock() function.
	if err := wait.PollImmediate(200*time.Millisecond, 2*time.Second, func() (bool, error) {
		l.lock14, err = net.ListenUnix("unix", &net.UnixAddr{Name: "@xtables", Net: "unix"})
		if err != nil {
			return false, nil
		}
		return true, nil
	}); err != nil {
		return nil, fmt.Errorf("failed to acquire old iptables lock: %v", err)
	}

	success = true
	return l, nil
}

func grabIptablesFileLock(f *os.File) error {
	return unix.Flock(int(f.Fd()), unix.LOCK_EX|unix.LOCK_NB)
}
