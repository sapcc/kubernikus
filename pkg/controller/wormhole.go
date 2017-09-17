package controller

type WormholeGenerator struct {
	Base
}

func NewWormholeGenerator(factories Factories, clients Clients) Controller {
	informers := factories.Kubernikus.Kubernikus().V1().Klusters().Informer()

	wg := &WormholeGenerator{
		NewBaseController(clients, informers),
	}

	wg.Controller = interface{}(wg).(Controller)

	return wg
}

