package flight

import "time"

type Instance interface {
	GetID() string
	GetName() string
	GetSecurityGroupNames() []string
	GetCreated() time.Time
	Erroring() bool
}
