/*
 *  Copyright (c) 2015 KingSoft.com, Inc. All Rights Reserved
 *  @file: monitor.go
 *  @brief:
 *  @author: suxiaolin(suxiaolin@kingsoft.com)
 *  @date: 2015/12/02 14:40:02
 *  @version: 0.0.1
 *  @history:
 */
package logmonitor

import (
	"errors"
	"sync"
	"time"
)

// 每个keyword的采集的指标值
type RuleData struct {
	// Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
	// 由Rule.Name赋值
	Name  string
	Value float64
	Tag   string
	Cycle int
}

type Monitor struct {
	// 对一个文件执行tail -f，新行会推送到每个rule，每个rule根据自己keyword进行正则匹配计算
	tail  *Tail
	group *Group
	// 保存所有rule名称，方便查找,Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
	rules map[string]struct{}
	// 指标结果的通道，Add函数中获取匹配行数后，构造指标放入这个通道。receive函数中读取该通道，累加匹配的行数，等待log获取该指标
	ruleDatas chan *RuleData
	// 按name组织指标值
	// key格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)，value为指标值实例
	sendData map[string]*RuleData
	stoped   bool
	senders  []Sender
	// 按name组织指标值
	// 采集的指标，key为ruleitem的Name，格式为 fmt.Sprintf("%s--------%s", "log.monitor", tag)
	Result      map[string]*RuleData
	monitorLock *sync.RWMutex
}

// 新建monitor实例，开启单个日志多个keyword匹配的日志监控
func NewMonitor(filename string, rules []*RuleItem, senders []Sender) (m *Monitor, err error) {
	m = &Monitor{}
	// 通过tail -f命令读取文件最新行，然后解析新行
	m.tail, err = NewTail(filename)
	if err != nil {
		return
	}
	m.group = NewGroup()
	var r *RuleItem
	m.rules = map[string]struct{}{}
	// m.Add内部调用的group.Add
	// group.Add根据每个RuleItem，生成Rule实例，并启动。Rule用于接收文件新行，解析匹配正则，计算匹配行数，推送匹配数
	// m.rules保存所有rule名称
	for _, r = range rules {
		// group中保存RuleItem
		err = m.Add(r)
		if err != nil {
			return
		}
		m.rules[r.Name] = struct{}{}
	}
	m.senders = senders
	// 通道，用于接收多个rule的指标值实例
	m.ruleDatas = make(chan *RuleData, 128)
	// 按name组织指标值
	m.sendData = map[string]*RuleData{}
	m.stoped = false
	// 按name组织指标值
	m.Result = map[string]*RuleData{}
	m.monitorLock = new(sync.RWMutex)
	// 开启日志监控，监控日志新行数，并推送给每个rule进行正则匹配。接收各个rule计算的匹配行数值，形成指标数据
	m.init()
	return
}

// 开启日志监控，监控日志新行数，并推送给每个rule进行正则匹配。接收各个rule计算的匹配行数值，形成指标数据
func (this *Monitor) init() {
	go func() {
		for !this.stoped {
			// 读取filepath的最新行，将最新行发送给每个rule的line通道
			this.monitor()
		}
	}()
	go func() {
		for !this.stoped {
			// 接收最新行数数据，保存到sendData。将sendData数据保存到各sender，并保存到this.Result供外层查询。重置sendData
			this.receive()
		}
	}()
}

// 读取filepath的最新行，将最新行发送给每个rule的line通道
func (this *Monitor) monitor() {
	defer func() {
		if e := recover(); e != nil {
			//catch a panic
		}
	}()
	var line *string
	var rule *Rule
	for !this.stoped {
		select {
		// 读取tail获取的此文件的最新Line
		case line = <-this.tail.Line:
			for _, rule = range this.group.Rules {
				// 新行发给每个rule
				rule.line <- line
			}
		}
	}
}

// 接收最新行数数据，保存到sendData。将sendData数据保存到各sender，并保存到this.Result供外层查询。重置sendData
func (this *Monitor) receive() {
	defer func() {
		if e := recover(); e != nil {
		}
	}()
	var ruledata *RuleData
	var ok bool
	var sender Sender
	var data *RuleData
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for !this.stoped {
		select {
		case data = <-this.ruleDatas:
			// 每个rule，每分钟都会向ruleDatas推送这此轮询匹配的行数的指标
			// 一直读取新指标数据通道，有的话就保存到this.sendData
			if ruledata, ok = this.sendData[data.Name]; ok {
				// data.Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
				// 有此name的数据，就累加
				this.sendData[data.Name].Value = this.sendData[data.Name].Value + ruledata.Value
			} else {
				// 没有此name，就创建
				this.sendData[data.Name] = data
			}
		case <-ticker.C:
			// 只要sendData有新数据，就保存到this.Result，每秒周期执行，做到尽快更新到Result
			// this.Result数据用于随时被查询，查询每个keywords本监控周期的值
			if len(this.sendData) != 0 {
				for _, sender = range this.senders {
					// 对每个sender都发送一下结果
					sender.Send(this.sendData)
					for _, sdata := range this.sendData {
						// 从sender读取数据，然后将数据保存到 this.Result
						// 上面调用了sender.Send，保存到sender；这里马上调用sender.GetMetricValue，读取sender
						// 这里sender好像并没什么作用，直接使用this.sendData也行
						this.SetValue(sdata.Name, sender.GetMetricValue()[sdata.Name])
					}
				}
				// 重置sendData
				this.sendData = map[string]*RuleData{}
			}
		}
	}
}

// 开启此rule对应的keywords的监控
// 监听rule的Value通道，此通道保证周期有行数数据推送过来，然后构造监控指标数据，发送给ruleDatas通道
func (this *Monitor) Add(r *RuleItem) (err error) {
	if _, ok := this.rules[r.Name]; !ok {
		// 开启此rule对新行的解析和正则匹配，即开启对此keyword的监控
		// Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
		err = this.group.Add(r.Name, r.Rule, r.Cycle, r.Type, r.Tag)
		if err != nil {
			return
		}
		this.rules[r.Name] = struct{}{}
		go func(rule *Rule) {
			defer func() {
				if e := recover(); e != nil {
					//catch a panic
				}
			}()
			var value float64
			var ok bool
			var data *RuleData
			data = &RuleData{}
			data.Name = rule.Name
			data.Value = 0
			data.Tag = rule.tag
			data.Cycle = rule.cycle
			for {
				select {
				// 读取匹配到的行数计数通道，这个通道周期会有数据推送过来
				case value, ok = <-rule.Value:
					if !ok {
						return
					}
					// 构造指标值，发送到ruleDatas
					data.Value = value
					this.ruleDatas <- data
				}
			}
		}(this.group.Rules[r.Name])
	}
	return
}

// 停止此keywords监控
func (this *Monitor) Del(name string) (err error) {
	if _, ok := this.rules[name]; !ok {
		err = errors.New("no exist thie name")
		return
	}
	this.group.Del(name)
	delete(this.rules, name)
	return
}

// 停止所有监控，停止所有sender
func (this *Monitor) Stop() {
	if !this.stoped {
		this.stoped = true
		for name, _ := range this.rules {
			this.Del(name)
		}
		close(this.ruleDatas)
		this.tail.Close()
		for _, sender := range this.senders {
			sender.Stop()
		}
	}
}

// 查询指标数据
func (this *Monitor) GetValue() map[string]*RuleData {
	this.monitorLock.RLock()
	defer this.monitorLock.RUnlock()
	return this.Result
}

// 更新指标数据
func (this *Monitor) SetValue(k string, data *RuleData) {
	this.monitorLock.Lock()
	defer this.monitorLock.Unlock()
	this.Result[k] = data
}
