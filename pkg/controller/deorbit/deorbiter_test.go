package deorbit

import (
	"context"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-kit/kit/log"
	th "github.com/gophercloud/gophercloud/testhelper"
	tc "github.com/gophercloud/gophercloud/testhelper/client"
	"github.com/stretchr/testify/assert"
	core_v1 "k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"

	kubernikus_v1 "github.com/sapcc/kubernikus/pkg/apis/kubernikus/v1"
)

var (
	kluster = &kubernikus_v1.Kluster{
		ObjectMeta: meta_v1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
		},
	}

	pvCinder0 = &core_v1.PersistentVolume{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pv-cinder0",
		},
		Spec: core_v1.PersistentVolumeSpec{
			PersistentVolumeSource: core_v1.PersistentVolumeSource{
				Cinder: &core_v1.CinderPersistentVolumeSource{
					VolumeID: "hase",
				},
			},
		},
	}

	pvCinder1 = &core_v1.PersistentVolume{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pv-cinder1",
		},
		Spec: core_v1.PersistentVolumeSpec{
			PersistentVolumeSource: core_v1.PersistentVolumeSource{
				Cinder: &core_v1.CinderPersistentVolumeSource{
					VolumeID: "maus",
				},
			},
		},
	}

	pvNFS = &core_v1.PersistentVolume{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pv-nfs",
		},
		Spec: core_v1.PersistentVolumeSpec{
			PersistentVolumeSource: core_v1.PersistentVolumeSource{
				NFS: &core_v1.NFSVolumeSource{},
			},
		},
	}

	pvcCinder0 = &core_v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pvc-cinder0",
		},
		Spec: core_v1.PersistentVolumeClaimSpec{
			VolumeName: "pv-cinder0",
		},
		Status: core_v1.PersistentVolumeClaimStatus{
			Phase: core_v1.ClaimBound,
		},
	}

	pvcCinder1 = &core_v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pvc-cinder1",
		},
		Spec: core_v1.PersistentVolumeClaimSpec{
			VolumeName: "pv-cinder1",
		},
		Status: core_v1.PersistentVolumeClaimStatus{
			Phase: core_v1.ClaimBound,
		},
	}

	pvcCinder2 = &core_v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pvc-cinder2",
		},
		Spec: core_v1.PersistentVolumeClaimSpec{},
		Status: core_v1.PersistentVolumeClaimStatus{
			Phase: core_v1.ClaimPending,
		},
	}

	pvcNFS = &core_v1.PersistentVolumeClaim{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "pvc-nfs",
		},
		Spec: core_v1.PersistentVolumeClaimSpec{
			VolumeName: "pv-nfs",
		},
		Status: core_v1.PersistentVolumeClaimStatus{
			Phase: core_v1.ClaimBound,
		},
	}

	svcLB0 *core_v1.Service = &core_v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "svc-lb0",
		},
		Spec: core_v1.ServiceSpec{
			Type: "LoadBalancer",
		},
	}

	svcLB1 = &core_v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "svc-lb1",
		},
		Spec: core_v1.ServiceSpec{
			Type: "LoadBalancer",
		},
	}

	svcCIP = &core_v1.Service{
		ObjectMeta: meta_v1.ObjectMeta{
			Name: "svc-cip",
		},
		Spec: core_v1.ServiceSpec{
			Type: "ClusterIP",
		},
	}
)

func TestIsServiceCleanupFinished(testing *testing.T) {
	type test_case struct {
		expected bool
		message  string
		objects  []runtime.Object
	}

	for i, t := range []test_case{
		{true, "true if no services with type LoadBalancer exist", []runtime.Object{svcCIP}},
		{false, "false if service with type LoadBalancer exists", []runtime.Object{svcLB0, svcLB1, svcCIP}},
	} {

		done := make(chan struct{})
		client := fake.NewSimpleClientset(t.objects...)
		logger := log.NewNopLogger()

		deorbiter := &ConcreteDeorbiter{kluster, done, client, logger, nil}
		finished, err := deorbiter.isServiceCleanupFinished()

		assert.Equal(testing, t.expected, finished, "Test %d failed: %v", i, t.message)
		assert.NoError(testing, err, "test %d failed", i)
	}
}

func TestIsPersistentVolumesCleanupFinished(testing *testing.T) {
	type test struct {
		message  string
		expected bool
		objects  []runtime.Object
	}

	for i, t := range []test{
		{"false if Cinder PVs remain", false, []runtime.Object{pvCinder0}},
		{"false if Cinder PVs remain", false, []runtime.Object{pvCinder0, pvNFS}},
		{"true if no Cinder PVs remain", true, []runtime.Object{pvNFS}},
		{"true if no PVs remain", true, []runtime.Object{}},
	} {

		done := make(chan struct{})
		client := fake.NewSimpleClientset(t.objects...)
		logger := log.NewNopLogger()

		th.SetupHTTP()
		defer th.TeardownHTTP()
		MockVolumeListResponse(testing)
		serviceClient := tc.ServiceClient()

		deorbiter := &ConcreteDeorbiter{kluster, done, client, logger, serviceClient}
		finished, err := deorbiter.isPersistentVolumesCleanupFinished()

		assert.Equal(testing, t.expected, finished, "Test %d failed: %v", i, t.message)
		assert.NoError(testing, err, "test %d failed", i)
	}
}

func MockVolumeListResponse(t *testing.T) {
	th.Mux.HandleFunc("/volumes/detail", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "GET")
		th.TestHeader(t, r, "X-Auth-Token", tc.TokenID)

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		r.ParseForm()
		marker := r.Form.Get("marker")
		switch marker {
		case "":
			fmt.Fprintf(w, `
  {
  "volumes": [
    {
      "volume_type": "lvmdriver-1",
      "created_at": "2015-09-17T03:35:03.000000",
      "bootable": "false",
      "name": "vol-001",
      "os-vol-mig-status-attr:name_id": null,
      "consistencygroup_id": null,
      "source_volid": null,
      "os-volume-replication:driver_data": null,
      "multiattach": false,
      "snapshot_id": null,
      "replication_status": "disabled",
      "os-volume-replication:extended_status": null,
      "encrypted": false,
      "os-vol-host-attr:host": "host-001",
      "availability_zone": "nova",
      "attachments": [{
        "server_id": "83ec2e3b-4321-422b-8706-a84185f52a0a",
        "attachment_id": "05551600-a936-4d4a-ba42-79a037c1-c91a",
        "attached_at": "2016-08-06T14:48:20.000000",
        "host_name": "foobar",
        "volume_id": "d6cacb1a-8b59-4c88-ad90-d70ebb82bb75",
        "device": "/dev/vdc",
        "id": "d6cacb1a-8b59-4c88-ad90-d70ebb82bb75"
      }],
      "id": "hase",
      "size": 75,
      "user_id": "ff1ce52c03ab433aaba9108c2e3ef541",
      "os-vol-tenant-attr:tenant_id": "304dc00909ac4d0da6c62d816bcb3459",
      "os-vol-mig-status-attr:migstat": null,
      "metadata": {"foo": "bar"},
      "status": "available",
      "description": null
    },
    {
      "volume_type": "lvmdriver-1",
      "created_at": "2015-09-17T03:32:29.000000",
      "bootable": "false",
      "name": "vol-002",
      "os-vol-mig-status-attr:name_id": null,
      "consistencygroup_id": null,
      "source_volid": null,
      "os-volume-replication:driver_data": null,
      "multiattach": false,
      "snapshot_id": null,
      "replication_status": "disabled",
      "os-volume-replication:extended_status": null,
      "encrypted": false,
      "os-vol-host-attr:host": null,
      "availability_zone": "nova",
      "attachments": [],
      "id": "maus",
      "size": 75,
      "user_id": "ff1ce52c03ab433aaba9108c2e3ef541",
      "os-vol-tenant-attr:tenant_id": "304dc00909ac4d0da6c62d816bcb3459",
      "os-vol-mig-status-attr:migstat": null,
      "metadata": {},
      "status": "available",
      "description": null
    }
  ],
	"volumes_links": [
	{
		"href": "%s/volumes/detail?marker=1",
		"rel": "next"
	}]
}
  `, th.Server.URL)
		case "1":
			fmt.Fprintf(w, `{"volumes": []}`)
		default:
			t.Fatalf("Unexpected marker: [%s]", marker)
		}
	})
}

func TestDeletePersistentVolumeClaims(testing *testing.T) {
	type test struct {
		message   string
		remaining int
		deleted   int
		objects   []runtime.Object
	}

	for i, t := range []test{
		{"deletes all Cinder PVs", 1, 2, []runtime.Object{pvCinder0, pvCinder1, pvcCinder0, pvcCinder1, pvcCinder2}},
		{"deletes only Cinder PVs", 2, 2, []runtime.Object{pvCinder0, pvCinder1, pvNFS, pvcCinder0, pvcCinder1, pvcCinder2, pvcNFS}},
	} {

		done := make(chan struct{})
		client := fake.NewSimpleClientset(t.objects...)
		logger := log.NewNopLogger()

		deorbiter := &ConcreteDeorbiter{kluster, done, client, logger, nil}
		deleted, err := deorbiter.DeletePersistentVolumeClaims()
		remaining, _ := client.CoreV1().PersistentVolumeClaims(meta_v1.NamespaceAll).List(context.Background(), meta_v1.ListOptions{})

		assert.Equal(testing, t.remaining, len(remaining.Items), "Test %d failed: %v", i, t.message)
		assert.Equal(testing, t.deleted, len(deleted), "Test %d failed: %v", i, t.message)
		assert.NoError(testing, err, "test %d failed", i)
	}
}

func TestDeleteServices(testing *testing.T) {
	type test struct {
		message   string
		remaining int
		deleted   int
		objects   []runtime.Object
	}

	for i, t := range []test{
		{"deletes all services of type LoadBalancer", 0, 2, []runtime.Object{svcLB0, svcLB1}},
		{"deletes only services of type LoadBalancer", 1, 2, []runtime.Object{svcCIP, svcLB0, svcLB1}},
	} {

		done := make(chan struct{})
		client := fake.NewSimpleClientset(t.objects...)
		logger := log.NewNopLogger()

		deorbiter := &ConcreteDeorbiter{kluster, done, client, logger, nil}
		deleted, err := deorbiter.DeleteServices()
		remaining, _ := client.CoreV1().Services(meta_v1.NamespaceAll).List(context.Background(), meta_v1.ListOptions{})

		assert.Equal(testing, t.remaining, len(remaining.Items), "Test %d failed: %v", i, t.message)
		assert.Equal(testing, t.deleted, len(deleted), "Test %d failed: %v", i, t.message)
		assert.NoError(testing, err, "test %d failed", i)
	}
}
