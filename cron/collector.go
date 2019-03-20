package cron

import (
	"time"

	"github.com/open-falcon/agent/funcs"
	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"
)

// 每秒刷新内存中的CPU和磁盘信息
func InitDataHistory() {
	for {
		funcs.UpdateCpuStat()
		funcs.UpdateDiskStats()
		time.Sleep(g.COLLECT_INTERVAL)
	}
}

// 周期执行内置监测
func Collect() {

	if !g.Config().Transfer.Enabled {
		return
	}

	if len(g.Config().Transfer.Addrs) == 0 {
		return
	}

	for _, v := range funcs.Mappers {
		go collect(int64(v.Interval), v.Fs)
	}
}

// 周期性执行监测，并过滤ignore指标
func collect(sec int64, fns []func() []*model.MetricValue) {
	t := time.NewTicker(time.Second * time.Duration(sec)).C
	for {
		<-t

		hostname, err := g.Hostname()
		if err != nil {
			continue
		}

		mvs := []*model.MetricValue{}
		ignoreMetrics := g.Config().IgnoreMetrics

		for _, fn := range fns {
			items := fn()
			if items == nil {
				continue
			}

			if len(items) == 0 {
				continue
			}

			for _, mv := range items {
				//先监测，然后再过滤监测指标
				if b, ok := ignoreMetrics[mv.Metric]; ok && b {
					continue
				} else {
					mvs = append(mvs, mv)
				}
			}
		}

		// 增加Endpoint、Step、Timestamp字段
		now := time.Now().Unix()
		for j := 0; j < len(mvs); j++ {
			mvs[j].Step = sec
			mvs[j].Endpoint = hostname
			mvs[j].Timestamp = now
		}

		g.SendToTransfer(mvs)

	}
}
