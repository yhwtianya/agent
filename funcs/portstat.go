package funcs

import (
	"fmt"
	"log"

	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"
	"github.com/toolkits/nux"
	"github.com/toolkits/slice"
)

// 执行端口监测
func PortMetrics() (L []*model.MetricValue) {

	// 获取要监测的端口
	reportPorts := g.ReportPorts()
	sz := len(reportPorts)
	if sz == 0 {
		return
	}

	// 获取所有监听端口
	allListeningPorts, err := nux.ListeningPorts()
	if err != nil {
		log.Println(err)
		return
	}

	for i := 0; i < sz; i++ {
		tags := fmt.Sprintf("port=%d", reportPorts[i])
		// 过滤监测的端口
		if slice.ContainsInt64(allListeningPorts, reportPorts[i]) {
			L = append(L, GaugeValue(g.NET_PORT_LISTEN, 1, tags))
		} else {
			L = append(L, GaugeValue(g.NET_PORT_LISTEN, 0, tags))
		}
	}

	return
}
