package deorbit

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	core_v1 "k8s.io/api/core/v1"
)

type InstrumentingDeorbiter struct {
	Deorbiter Deorbiter

	Latency    *prometheus.SummaryVec
	Total      *prometheus.CounterVec
	Successful *prometheus.CounterVec
	Failed     *prometheus.CounterVec
}

func (d *InstrumentingDeorbiter) DeletePersistentVolumeClaims() (deleted []core_v1.PersistentVolumeClaim, err error) {
	defer d.instrument("DeletePersistentVolumeClaims", time.Now(), err)
	return d.Deorbiter.DeletePersistentVolumeClaims()
}

func (d *InstrumentingDeorbiter) DeleteServices() (deleted []core_v1.Service, err error) {
	defer d.instrument("DeleteServices", time.Now(), err)
	return d.Deorbiter.DeleteServices()
}

func (d *InstrumentingDeorbiter) WaitForPersistentVolumeCleanup() (err error) {
	defer d.instrument("WaitForPersistentVolumeCleanup", time.Now(), err)
	return d.Deorbiter.WaitForPersistentVolumeCleanup()
}

func (d *InstrumentingDeorbiter) WaitForServiceCleanup() (err error) {
	defer d.instrument("WaitForServiceCleanup", time.Now(), err)
	return d.Deorbiter.WaitForServiceCleanup()
}

func (d *InstrumentingDeorbiter) SelfDestruct(reason SelfDestructReason) (err error) {
	defer d.instrument("SelfDestruct", time.Now(), err)
	return d.Deorbiter.SelfDestruct(reason)
}

func (d *InstrumentingDeorbiter) instrument(method string, begin time.Time, err error) {
	d.Latency.With(prometheus.Labels{"method": method}).Observe(time.Since(begin).Seconds())
	d.Total.With(prometheus.Labels{"method": method}).Add(1)

	if err != nil {
		d.Failed.With(prometheus.Labels{"method": method}).Add(1)
	} else {
		d.Successful.With(prometheus.Labels{"method": method}).Add(1)
	}
}

func (d *InstrumentingDeorbiter) IsAPIUnavailableTimeout() bool {
	return d.Deorbiter.IsAPIUnavailableTimeout()
}

func (d *InstrumentingDeorbiter) IsDeorbitHangingTimeout() bool {
	return d.Deorbiter.IsDeorbitHangingTimeout()
}
