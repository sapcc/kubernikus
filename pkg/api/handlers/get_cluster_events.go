package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
)

func NewGetClusterEvents(rt *api.Runtime) operations.GetClusterEventsHandler {
	return &getClusterEvents{Runtime: rt}
}

type getClusterEvents struct {
	*api.Runtime
}

func (d *getClusterEvents) Handle(params operations.GetClusterEventsParams, principal *models.Principal) middleware.Responder {
	eventsInterface := d.Kubernetes.Core().Events(d.Namespace)
	klusterName := qualifiedName(params.Name, principal.Account)
	selector := eventsInterface.GetFieldSelector(&klusterName, &d.Namespace, nil, nil)
	kEvents, err := eventsInterface.List(metav1.ListOptions{FieldSelector: selector.String()})
	if err != nil {
		return NewErrorResponse(&operations.GetClusterEventsDefault{}, 500, err.Error())
	}
	events := make([]*models.Event, 0, len(kEvents.Items))
	for _, ev := range kEvents.Items {
		events = append(events, &models.Event{
			FirstTimestamp: ev.FirstTimestamp.Time.String(),
			LastTimestamp:  ev.LastTimestamp.Time.String(),
			Message:        ev.Message,
			Reason:         ev.Reason,
			Count:          int64(ev.Count),
			Type:           ev.Type,
		})

	}

	return operations.NewGetClusterEventsOK().WithPayload(events)
}
