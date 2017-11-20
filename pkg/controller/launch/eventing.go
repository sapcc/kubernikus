package launch

import (
	api_v1 "k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/tools/record"

	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/controller/events"
)

type EventingPoolManager struct {
	PoolManager PoolManager
	Kluster     *v1.Kluster
	Recorder    record.EventRecorder
}

func (epm *EventingPoolManager) GetStatus() (status *PoolStatus, err error) {
	return epm.PoolManager.GetStatus()
}

func (epm *EventingPoolManager) SetStatus(status *PoolStatus) (err error) {
	return epm.PoolManager.SetStatus(status)
}

func (epm *EventingPoolManager) CreateNode() (id string, err error) {
	id, err = epm.PoolManager.CreateNode()

	if err == nil {
		epm.Recorder.Eventf(epm.Kluster, api_v1.EventTypeNormal, events.SuccessfullCreateNode, "Successfully created node %v", id)
	} else {
		epm.Recorder.Eventf(epm.Kluster, api_v1.EventTypeWarning, events.FailedCreateNode, "Failed to created node: %v", err)
	}

	return id, err
}

func (epm *EventingPoolManager) DeleteNode(id string, forceDelete bool) (err error) {
	err = epm.PoolManager.DeleteNode(id, false)

	if err == nil {
		epm.Recorder.Eventf(epm.Kluster, api_v1.EventTypeNormal, events.SuccessfullDeleteNode, "Successfully deleted node %v", id)
	} else {
		epm.Recorder.Eventf(epm.Kluster, api_v1.EventTypeWarning, events.FailedDeleteNode, "Failed to delete node: %v", err)
	}

	return
}

func (epm *EventingPoolManager) ResetNodeState(id string) (err error) {
	err = epm.PoolManager.ResetNodeState(id)

	if err == nil {
		epm.Recorder.Eventf(epm.Kluster, api_v1.EventTypeNormal, events.SuccessfullResetNodeState, "Successfully reset state of node %v", id)
	} else {
		epm.Recorder.Eventf(epm.Kluster, api_v1.EventTypeWarning, events.FailedResetNodeState, "Failed to reset state of node: %v", err)
	}

	return
}
