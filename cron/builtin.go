package cron

import (
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"
)

// 同步内置监测指标
func SyncBuiltinMetrics() {
	if g.Config().Heartbeat.Enabled && g.Config().Heartbeat.Addr != "" {
		go syncBuiltinMetrics()
	}
}

// 周期同步内置监测指标，url、port、procs、du四类监测指标
func syncBuiltinMetrics() {

	var timestamp int64 = -1
	var checksum string = "nil"

	duration := time.Duration(g.Config().Heartbeat.Interval) * time.Second

	for {
		time.Sleep(duration)

		var ports = []int64{}
		var paths = []string{}
		var procs = make(map[string]map[int]string)
		var urls = make(map[string]string)
		var logs = make(map[string]map[int]string)

		hostname, err := g.Hostname()
		if err != nil {
			continue
		}

		req := model.AgentHeartbeatRequest{
			Hostname: hostname,
			Checksum: checksum,
		}

		var resp model.BuiltinMetricResponse
		err = g.HbsClient.Call("Agent.BuiltinMetrics", req, &resp)
		if err != nil {
			log.Println("ERROR:", err)
			continue
		}

		if resp.Timestamp <= timestamp {
			continue
		}

		if resp.Checksum == checksum {
			continue
		}

		timestamp = resp.Timestamp
		checksum = resp.Checksum

		for _, metric := range resp.Metrics {

			if metric.Metric == g.URL_CHECK_HEALTH {
				arr := strings.Split(metric.Tags, ",")
				if len(arr) != 2 {
					continue
				}
				url := strings.Split(arr[0], "=")
				if len(url) != 2 {
					continue
				}
				stime := strings.Split(arr[1], "=")
				if len(stime) != 2 {
					continue
				}
				if _, err := strconv.ParseInt(stime[1], 10, 64); err == nil {
					urls[url[1]] = stime[1]
				} else {
					log.Println("metric ParseInt timeout failed:", err)
				}
			}

			if metric.Metric == g.NET_PORT_LISTEN {
				arr := strings.Split(metric.Tags, "=")
				if len(arr) != 2 {
					continue
				}

				if port, err := strconv.ParseInt(arr[1], 10, 64); err == nil {
					ports = append(ports, port)
				} else {
					log.Println("metrics ParseInt failed:", err)
				}

				continue
			}

			if metric.Metric == g.DU_BS {
				arr := strings.Split(metric.Tags, "=")
				if len(arr) != 2 {
					continue
				}

				paths = append(paths, strings.TrimSpace(arr[1]))
				continue
			}

			if metric.Metric == g.PROC_NUM {
				arr := strings.Split(metric.Tags, ",")

				tmpMap := make(map[int]string)

				for i := 0; i < len(arr); i++ {
					if strings.HasPrefix(arr[i], "name=") {
						tmpMap[1] = strings.TrimSpace(arr[i][5:])
					} else if strings.HasPrefix(arr[i], "cmdline=") {
						tmpMap[2] = strings.TrimSpace(arr[i][8:])
					}
				}

				procs[metric.Tags] = tmpMap
			}

			if metric.Metric == g.LOG_MONITOR {
				// 从tags中解析filepath和keywords，保存到logs，logs保存多个logmonitor，key为tags字符串。logs最后保存到全局变量reportLogs
				// tags: filepath=/opt/deploy/tiantian/log/tiantian.log,keywords=\[W\]
				arr := strings.Split(metric.Tags, ",")
				if len(arr) != 2 {
					continue
				}

				tmpMap := make(map[int]string)

				for i := 0; i < len(arr); i++ {
					if strings.HasPrefix(arr[i], "filepath=") {
						tmpMap[1] = strings.TrimSpace(arr[i][9:])
					} else if strings.HasPrefix(arr[i], "keywords=") {
						tmpMap[2] = strings.TrimSpace(arr[i][9:])
					}
				}

				logs[metric.Tags] = tmpMap
			}
		}

		// 将监测指标保存到全局变量
		g.SetReportUrls(urls)
		g.SetReportPorts(ports)
		g.SetReportProcs(procs)
		g.SetDuPaths(paths)
		g.SetReportLogs(logs)

	}
}
