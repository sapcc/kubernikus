// This command is a helper functions that looks up dependency revisions
// for the packages used by the go-swagger generated code
//
// It lists the dependencies used be a given package and looks
// for matching entries in go-swaggers Gopkg.lock file
package main

import (
	"bufio"
	"bytes"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os/exec"
	"regexp"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/spf13/pflag"
)

type Project struct {
	Name     string
	Branch   string
	Packages []string
	Revision string
}

type Deps struct {
	Projects []Project
}

var (
	blacklist []string
	version   string
)

func main() {
	pflag.StringVarP(&version, "version", "v", "0.12.0", "go-swagger version")
	pflag.StringArrayVarP(&blacklist, "blacklist", "b", []string{"golang.org/x/net", "golang.org/x/text"}, "don't consider the given dependencies")

	pflag.Parse()
	packages := flag.Args()
	if len(packages) == 0 {
		packages = []string{"github.com/sapcc/kubernikus/pkg/api/rest/operations"}
	}

	url := fmt.Sprintf("https://raw.githubusercontent.com/go-swagger/go-swagger/%s/Gopkg.lock", version)
	log.Println("Fetching ", url)

	resp, err := http.Get(url)
	if err != nil {
		log.Fatal(err)
	}
	if resp.StatusCode >= 400 {
		log.Fatalf("Failed to fetch Gopkg.lock: %s", resp.Status)
	}

	var deps Deps

	_, err = toml.DecodeReader(resp.Body, &deps)
	if err != nil {
		log.Fatal(err)
	}

	cmd := exec.Command("go", append([]string{"list", "-f", `{{ join .Deps "\n" }}`}, packages...)...)
	log.Printf("Running %v", cmd.Args)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatal(err)
	}
	result := map[string]string{}
	scanner := bufio.NewScanner(bytes.NewReader(output))
	re := regexp.MustCompile(`kubernikus/vendor/(.+)`)
	for scanner.Scan() {
		dep := scanner.Text()
		if matches := re.FindStringSubmatch(dep); matches != nil {
			for _, p := range deps.Projects {
				if blacklisted(p.Name) {
					continue
				}
				if strings.HasPrefix(matches[1], p.Name) {
					result[p.Name] = p.Revision
					break
				}
			}
		}
	}
	fmt.Printf("# Dependencies extracted from go-swagger %s\n", version)
	for pkg, rev := range result {
		fmt.Printf("- package: %s\n", pkg)
		fmt.Printf("  version: %s\n", rev)
	}

}

func blacklisted(dep string) bool {
	for _, entry := range blacklist {
		if entry == dep {
			return true
		}
	}
	return false
}
