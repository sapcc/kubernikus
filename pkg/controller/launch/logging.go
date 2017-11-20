package launch

import (
	"strconv"
	"time"

	"github.com/go-kit/kit/log"
)

type LoggingPoolManager struct {
	PoolManager PoolManager
	Logger      log.Logger
}

func (npm *LoggingPoolManager) GetStatus() (status *PoolStatus, err error) {
	defer func(begin time.Time) {
		npm.Logger.Log(
			"msg", "read status",
			"running", status.Running,
			"starting", status.Starting,
			"stopping", status.Stopping,
			"needed", status.Needed,
			"unneeded", status.UnNeeded,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return npm.PoolManager.GetStatus()
}

func (npm *LoggingPoolManager) SetStatus(status *PoolStatus) (err error) {
	defer func(begin time.Time) {
		npm.Logger.Log(
			"msg", "wrote status",
			"running", status.Running,
			"starting", status.Starting,
			"stopping", status.Stopping,
			"needed", status.Needed,
			"unneeded", status.UnNeeded,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return npm.PoolManager.SetStatus(status)
}

func (npm *LoggingPoolManager) CreateNode() (id string, err error) {
	defer func(begin time.Time) {
		npm.Logger.Log(
			"msg", "created node",
			"node", id,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return npm.PoolManager.CreateNode()
}

func (npm *LoggingPoolManager) DeleteNode(id string, forceDelete bool) (err error) {
	defer func(begin time.Time) {
		npm.Logger.Log(
			"msg", "deleted node",
			"node", id,
			"took", time.Since(begin),
			"err", err,
			"force", strconv.FormatBool(forceDelete),
		)
	}(time.Now())
	return npm.PoolManager.DeleteNode(id, false)
}

func (npm *LoggingPoolManager) ResetNodeState(id string) (err error) {
	defer func(begin time.Time) {
		npm.Logger.Log(
			"msg", "reset node state",
			"node", id,
			"took", time.Since(begin),
			"err", err,
		)
	}(time.Now())
	return npm.PoolManager.ResetNodeState(id)
}
