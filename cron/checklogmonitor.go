package cron

import (
	"strconv"
	"strings"
	"time"

	"github.com/open-falcon/agent/funcs/logmonitor"
	"github.com/open-falcon/agent/g"
)

func CheckLogMonitor() {
	go checkLogMonitor()
}

// 周期检查过期log.monitor
func checkLogMonitor() {
	interval := g.Config().Transfer.Interval
	for {
		logTimeMap := g.GetLogTimeMap()
		now := time.Now().Unix()
		for key, m := range logTimeMap {
			keyInfo := strings.Split(key, "::::::")
			t, err := strconv.ParseInt(keyInfo[1], 10, 64)
			if err != nil {
				continue
			}
			if (now - t) >= 70 {
				// 超过70s没更新采集任务就删除掉
				m.Stop()
				// 在logFile中删除
				logmonitor.UpdateLogFile(keyInfo[0])
				// 在logTimeMap中删除
				g.UpdateLogTimeMap(key)
			}
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}
