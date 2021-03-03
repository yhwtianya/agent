package logmonitor

import (
	"os"
	"sync"
)

// Falcon实现了Sender接口，每个filepath对应一个sender
type Falcon struct {
	sync.RWMutex
	// 本机主机名，其实并没使用
	endpoint string
	stoped   bool
	// 保存每个keywords的指标
	data map[string]*RuleData
}

func (this *Falcon) Init() (err error) {
	this.Lock()
	defer this.Unlock()
	this.endpoint, err = os.Hostname()
	if err != nil {
		return
	}
	this.stoped = false
	this.data = make(map[string]*RuleData)
	return
}

// 保存指标
func (this *Falcon) Send(data map[string]*RuleData) {
	this.Lock()
	defer this.Unlock()
	this.data = data
}

// 返回指标
func (this *Falcon) GetMetricValue() map[string]*RuleData {
	this.RLock()
	defer this.RUnlock()
	return this.data
}

func (this *Falcon) Stop() {
	this.stoped = true
}
