package launch

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type InstrumentingPoolManager struct {
	PoolManager PoolManager

	Latency    *prometheus.SummaryVec
	Total      *prometheus.CounterVec
	Successful *prometheus.CounterVec
	Failed     *prometheus.CounterVec
}

func (pm *InstrumentingPoolManager) GetStatus() (status *PoolStatus, err error) {
	defer func(begin time.Time) {
		pm.Latency.With(
			prometheus.Labels{
				"method": "GetStatus",
			}).Observe(time.Since(begin).Seconds())

		pm.Total.With(
			prometheus.Labels{
				"method": "GetStatus",
			}).Add(1)

		if err != nil {
			pm.Failed.With(
				prometheus.Labels{
					"method": "GetStatus",
				}).Add(1)
		} else {
			pm.Successful.With(
				prometheus.Labels{
					"method": "GetStatus",
				}).Add(1)
		}
	}(time.Now())
	return pm.PoolManager.GetStatus()
}

func (pm *InstrumentingPoolManager) SetStatus(status *PoolStatus) (err error) {
	defer func(begin time.Time) {
		pm.Latency.With(
			prometheus.Labels{
				"method": "SetStatus",
			}).Observe(time.Since(begin).Seconds())

		pm.Total.With(
			prometheus.Labels{
				"method": "SetStatus",
			}).Add(1)

		if err != nil {
			pm.Failed.With(
				prometheus.Labels{
					"method": "SetStatus",
				}).Add(1)
		} else {
			pm.Successful.With(
				prometheus.Labels{
					"method": "SetStatus",
				}).Add(1)
		}
	}(time.Now())
	return pm.PoolManager.SetStatus(status)
}

func (pm *InstrumentingPoolManager) CreateNode() (id string, err error) {
	defer func(begin time.Time) {
		pm.Latency.With(
			prometheus.Labels{
				"method": "CreateNode",
			}).Observe(time.Since(begin).Seconds())

		pm.Total.With(
			prometheus.Labels{
				"method": "CreateNode",
			}).Add(1)

		if err != nil {
			pm.Failed.With(
				prometheus.Labels{
					"method": "CreateNode",
				}).Add(1)
		} else {
			pm.Successful.With(
				prometheus.Labels{
					"method": "CreateNode",
				}).Add(1)
		}
	}(time.Now())

	return pm.PoolManager.CreateNode()
}

func (pm *InstrumentingPoolManager) DeleteNode(id string) (err error) {
	defer func(begin time.Time) {
		pm.Latency.With(
			prometheus.Labels{
				"method": "DeleteNode",
			}).Observe(time.Since(begin).Seconds())

		pm.Total.With(
			prometheus.Labels{
				"method": "DeleteNode",
			}).Add(1)

		if err != nil {
			pm.Failed.With(
				prometheus.Labels{
					"method": "DeleteNode",
				}).Add(1)
		} else {
			pm.Successful.With(
				prometheus.Labels{
					"method": "DeleteNode",
				}).Add(1)
		}
	}(time.Now())

	return pm.PoolManager.DeleteNode(id)
}
