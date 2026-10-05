package service

import (
	"context"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var upstreamTaskOnce sync.Once

func StartUpstreamRefreshTask() {
	upstreamTaskOnce.Do(func() {
		if !common.IsMasterNode {
			return
		}
		go func() {
			for {
				config, err := model.GetUpstreamConfig()
				interval := 30 * time.Minute
				if err == nil && config.IntervalMinutes > 0 {
					interval = time.Duration(config.IntervalMinutes) * time.Minute
				}
				time.Sleep(interval)
				if err == nil && config.IntervalMinutes > 0 {
					_ = RefreshAllUpstreams(context.Background())
				}
			}
		}()
	})
}
