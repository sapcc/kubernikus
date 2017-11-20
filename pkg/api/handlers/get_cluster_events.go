package handlers

import (
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/sapcc/kubernikus/pkg/api"
	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/api/rest/operations"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewGetClusterEvents(rt *api.Runtime) operations.GetClusterEventsHandler {
	return &getClusterEvents{Runtime: rt}
}

type getClusterEvents struct {
	*api.Runtime
}

func (d *getClusterEvents) Handle(params operations.GetClusterEventsParams, principal *models.Principal) middleware.Responder {
	eventsInterface := d.Kubernetes.Core().Events("")
	klusterName := qualifiedName(params.Name, principal.Account)
	selector := eventsInterface.GetFieldSelector(&klusterName, &d.Namespace, nil, nil)
	kEvents, err := eventsInterface.List(metav1.ListOptions{FieldSelector: selector.String()})
	if err != nil {
		return NewErrorResponse(&operations.GetClusterEventsDefault{}, 500, err.Error())
	}
	events := make([]*models.Event, 0, len(kEvents.Items))
	for _, ev := range kEvents.Items {
		events = append(events, &models.Event{
			FirstTimestamp: strfmt.DateTime(ev.FirstTimestamp.Time),
			LastTimestamp:  strfmt.DateTime(ev.LastTimestamp.Time),
			Message:        ev.Message,
			Reason:         ev.Reason,
			Count:          int64(ev.Count),
		})

	}

	return operations.NewGetClusterEventsOK().WithPayload(events)
}
