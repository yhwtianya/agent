package funcs

import (
	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"
)

// 保存同一轮询间隔的监测函数
type FuncsAndInterval struct {
	Fs       []func() []*model.MetricValue
	Interval int
}

// 保存所有监测函数
var Mappers []FuncsAndInterval

// 构建所有监测函数,保存到Mappers
// 轮询间隔都是使用的 g.Config().Transfer.Interval
func BuildMappers() {
	interval := g.Config().Transfer.Interval
	Mappers = []FuncsAndInterval{
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				AgentMetrics,
				CpuMetrics,
				NetMetrics,
				KernelMetrics,
				LoadAvgMetrics,
				MemMetrics,
				DiskIOMetrics,
				IOStatsMetrics,
				NetstatMetrics,
				ProcMetrics,
				UdpMetrics,
			},
			Interval: interval,
		},
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				DeviceMetrics,
			},
			Interval: interval,
		},
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				PortMetrics,
				SocketStatSummaryMetrics,
			},
			Interval: interval,
		},
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				DuMetrics,
			},
			Interval: interval,
		},
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				UrlMetrics,
			},
			Interval: interval,
		},
		FuncsAndInterval{
			Fs: []func() []*model.MetricValue{
				LogMetrics,
			},
			Interval: interval,
		},
	}
}
