package funcs

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/open-falcon/agent/funcs/logmonitor"
	"github.com/open-falcon/agent/g"
	"github.com/open-falcon/common/model"
)

func LogMetrics() (L []*model.MetricValue) {

	// tags: filepath=/opt/deploy/tiantian/log/tiantian.log,keywords=\[W\]
	// reportLogs保存了所有log监控指标，外层key为tags字符串，内层key取值为1和2，分别对应filepath和keywords的值
	// reportLogs是从hbs最新同步过来的log.monitor监控项
	reportLogs := g.ReportLogs()
	sz := len(reportLogs)
	if sz == 0 {
		return
	}

	var filepath string
	var keywords string

	for tags, m := range reportLogs {
		// 获取路径和keyword
		for key, val := range m {
			if key == 1 {
				filepath = val
			} else if key == 2 {
				keywords = val
			}
		}

		// logFile即logOldFile，记录了之前所有log monitor， 外层key为filepath，内层key为keywords，value为对应Monitor
		logOldFile := logmonitor.GetLogFile()
		var keys []string
		// 如果此file配置了多个监控项，keys保存所有的keywords
		keys = append(keys, keywords)
		if v, existed := logOldFile[filepath]; existed {
			// 存在此filepath监控
			if _, existed := v[keywords]; !existed {
				// 不存在此keywords监控
				for k, m := range v {
					// 将此文件的其他要监控的keywords保存，停止此文件监控，后面启动新的文件监控
					keys = append(keys, k)
					// 停止监控
					m.Stop()
					// lm，保存了所有log monitor 外层key为filepath::::::timestamp，value为对应Monitor
					lm := g.GetLogTimeMap()
					for lmkey, _ := range lm {
						if strings.Contains(lmkey, filepath) {
							// 由于此文件更新了新keyword，在logTimeMap中删除，后面会重新保存到logTimeMap中
							// 由于此文件仍需监控，logFile中并不删除。但是如果keywords已经被用户删除，是需要在logFile中删除的
							// logFile删除过期keywords的行为是在CheckLogMonitor函数中进行的
							g.UpdateLogTimeMap(lmkey)
						}
					}
				}
			} else {
				// logFile记录了此keywords监控
				// 需要根据当前时间戳刷新key，重新保存到logTimeMap中
				timestamp := time.Now().Unix()
				// lm，保存了所有log monitor 外层key为filepath，value为对应Monitor
				lm := g.GetLogTimeMap()
				for key, m := range lm {
					keyInfo := strings.Split(key, "::::::")
					if keyInfo[0] == filepath {
						// logTimeMap中删除此filepath的记录
						g.UpdateLogTimeMap(key)
						// 根据当前timestamp更新key名称，logTimeMap添加此filepath的记录
						newKey := fmt.Sprintf("%s::::::%d", filepath, timestamp)
						g.SetLogTimeMap(newKey, m)
					}
				}
				// 继续遍历reportLogs
				continue
			}
		}
		// rules是RuleItem的切片，代表指标，每个keywords对应一个rule
		rules := []*logmonitor.RuleItem{}
		senders := []logmonitor.Sender{}
		for _, key := range keys {
			tagInfo := strings.Split(tags, ",")
			tag := fmt.Sprintf("%s,keywords=%s", tagInfo[0], key)
			key = strings.Replace(key, "\\\\", "\\", -1)
			ruleItem := &logmonitor.RuleItem{}
			ruleItem.Cycle = 60
			ruleItem.Name = fmt.Sprintf("%s--------%s", "log.monitor", tag)
			ruleItem.Rule = key
			// 类型为总数
			ruleItem.Type = "sum"
			ruleItem.Tag = tag
			// 每个keywords是一个rule
			rules = append(rules, ruleItem)
		}

		// Falcon实现了Sender接口
		falcon := &logmonitor.Falcon{}
		err := falcon.Init()
		if err != nil {
			log.Println("init sender failed error:", err.Error())
			os.Exit(2)
		}
		// 一个filepath对应一个sender
		senders = append(senders, falcon)
		// 一个filepath对应一个monitor
		monitor, err := logmonitor.NewMonitor(filepath, rules, senders)
		if err != nil {
			log.Println("init monitor error:", err.Error(), monitor)
			os.Exit(2)
			return
		}
		// 将此filepath的此keywords加入到logFile
		logmonitor.SetLogFile(filepath, keywords, monitor)
		timestamp := time.Now().Unix()
		// 更新key名称，logTimeMap添加此filepath的记录
		key := fmt.Sprintf("%s::::::%d", filepath, timestamp)
		g.SetLogTimeMap(key, monitor)
	}
	// Lmap，保存了所有log monitor 外层key为filepath，value为对应Monitor
	Lmap := g.GetLogTimeMap()
	// 获取所有log采集的指标
	for _, m := range Lmap {
		for k, d := range m.GetValue() {
			// key 格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
			key := strings.Split(k, "--------")
			L = append(L, GaugeValue(key[0], d.Value, d.Tag))
		}
	}
	return
}
