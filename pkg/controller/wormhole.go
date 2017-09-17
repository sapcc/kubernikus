package controller

import (
	"fmt"
	"time"

	"github.com/golang/glog"
	"github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
	"github.com/sapcc/kubernikus/pkg/client/openstack"
)

type WormholeGenerator struct {
	Base
}

type State struct {
	key     string
	kluster *v1.Kluster
	node    *openstack.Node
	message string
	err     error
}

type Transition func(*State) (Transition, error)

func NewWormholeGenerator(factories Factories, clients Clients) Controller {
	informers := factories.Kubernikus.Kubernikus().V1().Klusters().Informer()

	wg := &WormholeGenerator{
		NewBaseController(clients, informers),
	}

	wg.Controller = interface{}(wg).(Controller)

	return wg
}

func (wg *WormholeGenerator) reconcile(key string) error {
	var err error
	state := &State{key: key}
	transition := wg.start

	for transition != nil && err == nil {
		transition, err = transition(state)
		if state.message != "" {
			glog.V(5).Infof("[%v] %v", key, state.message)
			state.message = ""
		}
	}

	return err
}

func (wg *WormholeGenerator) start(state *State) (Transition, error) {
	obj, exists, err := wg.informer.GetIndexer().GetByKey(state.key)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch key %s from cache: %s", state.key, err)
	}

	if !exists {
		state.message = "Kluster deleted in the meantime"
		return nil, nil
	}

	state.kluster = obj.(*v1.Kluster)

	return wg.findOrCreateWormhole, nil
}

func (wg *WormholeGenerator) findOrCreateWormhole(state *State) (Transition, error) {
	wormhole, err := wg.Clients.Openstack.GetWormhole(state.kluster)
	if err != nil {
		return nil, fmt.Errorf("Couldn't get wormhole VM: %v", err)
	}

	if wormhole == nil {
		state.message = "Wormhole does not exist. Need to create it."
		return wg.createWormhole, nil
	}

	state.node = wormhole
	state.message = fmt.Sprintf("Found wormhole: %v", wormhole.Name)
	return wg.checkWormhole, nil
}

func (wg *WormholeGenerator) checkWormhole(state *State) (Transition, error) {
	if state.node.Ready() {
		state.message = "Wormhole ok"
		return nil, nil
	}

	if time.Since(state.node.Created) > 5*time.Minute {
		state.message = "Wormhole is hanging. Trying to repair."
		return wg.repairWormhole, nil
	}

	return nil, fmt.Errorf("Wormhole is not ready")
}

func (wg *WormholeGenerator) repairWormhole(state *State) (Transition, error) {
	err := wg.Clients.Openstack.DeleteNode(state.kluster, state.node.ID)
	if err != nil {
		return nil, fmt.Errorf("Couldn't repair wormhole %v: %v", state.node.Name, err)
	}
	state.message = fmt.Sprintf("Terminated wormhole %v", state.node.Name)
	return wg.findOrCreateWormhole, nil
}

func (wg *WormholeGenerator) createWormhole(state *State) (Transition, error) {
	name, err := wg.Clients.Openstack.CreateWormhole(state.kluster)
	if err != nil {
		return nil, err
	}

	state.message = fmt.Sprintf("Wormhole %v ceated", name)
	return wg.findOrCreateWormhole, nil
}
