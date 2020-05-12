package servicing

import (
	"testing"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/assert"

	"github.com/sapcc/kubernikus/pkg/api/models"
	"github.com/sapcc/kubernikus/pkg/controller/nodeobservatory"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/coreos"
	"github.com/sapcc/kubernikus/pkg/controller/servicing/flatcar"
	kubernikusfake "github.com/sapcc/kubernikus/pkg/generated/clientset/fake"
)

func TestServicingControllerReconcile(t *testing.T) {
	Now = func() time.Time { return time.Date(2019, 2, 3, 4, 0, 0, 0, time.UTC) }

	rec := Now().Add(-1 * time.Minute)
	pre := Now().Add(-1 * ServiceInterval).Add(-1 * time.Second)
	now := Now()

	type test struct {
		message         string
		options         *FakeKlusterOptions
		expectedDrain   bool
		expectedReboot  bool
		expectedReplace bool
	}
	for _, subject := range []test{
		//
		// Test Kluster Phase
		//
		{
			message: "Running klusters should be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   true,
			expectedReboot:  false,
			expectedReplace: true,
		},
		{
			message: "Creating klusters should not be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseCreating,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false,
		},
		{
			message: "Pending klusters should not be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhasePending,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false},
		{
			message: "Terminating klusters should not be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseTerminating,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false,
		},

		//
		//  Test Service Interval
		//
		{
			message: "Never serviced klusters should be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   true,
			expectedReboot:  false,
			expectedReplace: true},
		{
			message: "Klusters serviced recently should not be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: &rec,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false,
		},
		{
			message: "Klusters serviced longer than service interval ago should be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: &pre,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   true,
			expectedReboot:  false,
			expectedReplace: true,
		},
		{
			message: "Klusters serviced twice in a row should not be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: &now,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false,
		},

		//
		// Test Unhealthy Klusters
		//
		{
			message: "Unhealthy klusters should not be reconciled",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         false,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false,
		},

		//
		// Test Replacement
		//
		{
			message: "Nodes with outdated kubelet and OS should be replaced",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: true,
						Size:                1,
					},
				},
			},
			expectedDrain:   true,
			expectedReboot:  false,
			expectedReplace: true,
		},
		{
			message: "Nodes with outdate OS should be rebooted",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      true,
						NodeKubeletOutdated: false,
						Size:                1,
					},
				},
			},
			expectedDrain:   true,
			expectedReboot:  true,
			expectedReplace: false,
		},
		{
			message: "Up-to-date Node should neither be rebooted nor be replaced",
			options: &FakeKlusterOptions{
				Phase:       models.KlusterPhaseRunning,
				LastService: nil,
				NodePools: []FakeNodePoolOptions{
					{
						AllowReboot:         true,
						AllowReplace:        true,
						NodeHealthy:         true,
						NodeOSOutdated:      false,
						NodeKubeletOutdated: false,
						Size:                1,
					},
				},
			},
			expectedDrain:   false,
			expectedReboot:  false,
			expectedReplace: false,
		},
	} {
		t.Run(string(subject.message), func(t *testing.T) {
			kluster, nodes := NewFakeKluster(subject.options)
			logger := log.With(TestLogger(), "controller", "servicing")

			mockCycler := &MockLifeCycler{}
			mockCycler.On("Reboot", nodes[0]).Return(nil).Times(0)
			mockCycler.On("Drain", nodes[0]).Return(nil).Times(0)
			mockCycler.On("Replace", nodes[0]).Return(nil).Times(0)
			mockCycler.On("Uncordon", nodes[0]).Return(nil).Times(0)

			var cycler LifeCycler = &LoggingLifeCycler{
				Logger:     log.With(logger, "kluster", kluster.Spec.Name, "project", kluster.Account()),
				LifeCycler: mockCycler,
			}

			lifecyclers := &MockLifeCyclerFactory{}
			lifecyclers.On("Make", kluster).Return(cycler, nil)

			listers := &NodeListerFactory{
				Logger:          logger,
				NodeObservatory: nodeobservatory.NewFakeController(kluster, nodes...),
				CoreOSVersion:   coreos.NewFakeVersion(t, "2023.4.0"),
				CoreOSRelease:   coreos.NewFakeRelease(t, "2023.4.0"),
				FlatcarVersion:  flatcar.NewFakeVersion(t, "2303.4.0"),
				FlatcarRelease:  flatcar.NewFakeRelease(t, "2303.4.0"),
			}

			reconcilers := &KlusterReconcilerFactory{
				Logger:            logger,
				ListerFactory:     listers,
				LifeCyclerFactory: lifecyclers,
				KlusterLister:     NewFakeKlusterLister(kluster),
				KubernikusClient:  kubernikusfake.NewSimpleClientset(kluster).Kubernikus(),
			}

			controller := &Controller{
				Logger:     logger,
				Reconciler: reconcilers,
			}

			_, err := controller.Reconcile(kluster)
			if subject.expectedDrain {
				mockCycler.AssertCalled(t, "Drain", nodes[0])
			} else {
				mockCycler.AssertNotCalled(t, "Drain")
			}

			if subject.expectedReboot {
				mockCycler.AssertCalled(t, "Reboot", nodes[0])
			} else {
				mockCycler.AssertNotCalled(t, "Reboot")
			}

			if subject.expectedReplace {
				mockCycler.AssertCalled(t, "Replace", nodes[0])
			} else {
				mockCycler.AssertNotCalled(t, "Replace")
			}
			assert.NoError(t, err)
		})
	}
}
