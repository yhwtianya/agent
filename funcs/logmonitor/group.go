/*
 *  Copyright (c) 2015 KingSoft.com, Inc. All Rights Reserved
 *  @file: group.go
 *  @brief:
 *  @author: suxiaolin(suxiaolin@kingsoft.com)
 *  @date: 2015/12/06 18:12:18
 *  @version: 0.0.1
 *  @history:
 */

package logmonitor

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"time"
)

// 用于管理一个Monitor的所有Rule。Add负责加入并开始Rule，Del负责停止并删除Rule
type Group struct {
	// 保存了Monitor关联的所有rule，每个keywords是一个rule
	Rules map[string]*Rule
}

func NewGroup() *Group {
	g := &Group{}
	g.Rules = map[string]*Rule{}
	return g
}

// 添加rule，并启动rule，rule会接收新行，进行正则匹配，并周期推送匹配的行数
// Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
func (this *Group) Add(name string, sregexp string, cycle int, t string, tag string) (err error) {
	_, ok := this.Rules[name]
	if ok {
		err = errors.New("name is exist")
		return
	}
	// 新建rule，并启动接收新行协程，然后进行正则匹配，然后周期推送此rule匹配的行数数据到rule.Value通道
	r, err := NewRule(name, sregexp, cycle, t, tag)
	if err != nil {
		return err
	}
	this.Rules[name] = r
	return
}

// 停止此rule接收新行，停止解析匹配，删除rule
func (this *Group) Del(name string) {
	_, ok := this.Rules[name]
	if !ok {
		return
	}
	// 停止rule对新行的解析
	this.Rules[name].Stop()
	delete(this.Rules, name)
}

// 代表一个指标值，log monitor可以对同一文件配置多条监控，每个keywords对应rule
type RuleItem struct {
	// Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
	Name string
	// 正则表达式
	Rule string
	// 类型强制为sum，即总数
	Type string
	//tags: filepath=/opt/deploy/tiantian/log/tiantian.log,keywords=\[W\]
	Tag string
	// 周期
	Cycle int
}

//reuse var to reduce memory use
type tmpVar struct {
	index     int
	key       string
	value     string
	value_f64 float64
	err       error
}

// 每个keywords对一个Rule
type Rule struct {
	// Name格式为fmt.Sprintf("%s--------%s", "log.monitor", tag)
	Name string
	// 一般为sum和avg，log一般强制为sum
	Type string
	// 通道，receive函数会周期调用send函数，向此通道推送匹配的总行数。在monitor的Add函数里被循环读取
	// 即使日志文件不存在，行数采集失败，send也会推送0，保证数据连续性
	Value chan float64
	// 累加个数
	value float64
	count float64
	// 周期
	cycle int
	// tags: filepath=/opt/deploy/tiantian/log/tiantian.log,keywords=\[W\]
	tag string
	// 通道，用于接收tail获取的新行内容
	line chan *string
	// keywords编译
	regexp *regexp.Regexp
	// 子匹配内容，第一个为整体匹配，一般为空
	subexpNames []string
	// 匹配的子匹配字符串
	matches  []string
	oneMatch string
	// keywords是否含有子匹配
	subMatch bool
	out      map[string]interface{}
	tmpvar   *tmpVar
}

// 新建rule，并启动接收新行协程，然后进行正则匹配，然后周期推送此rule匹配的行数数据到rule.Value通道
func NewRule(name string, sregexp string, cycle int, t string, tag string) (*Rule, error) {
	rule := &Rule{}
	var err error
	keyword := fmt.Sprintf(`%s`, sregexp)
	if rule.regexp, err = regexp.Compile(keyword); err != nil {
		return rule, err
	}
	rule.Name = name
	rule.Type = t
	rule.Value = make(chan float64, 64)
	rule.value = 0
	rule.count = 0
	rule.cycle = cycle
	rule.tag = tag
	rule.line = make(chan *string, LINE_BUFFER_SIZE)
	// 检查keyword是否含有子匹配，即括号表达式
	rule.subexpNames = rule.regexp.SubexpNames()
	if len(rule.subexpNames) == 1 {
		rule.subMatch = false
	} else {
		rule.subMatch = true
	}
	rule.tmpvar = &tmpVar{}

	// 读取line通道，获取文件最新行，按正则条件匹配，累加匹配数。内部周期调用send发送累加数
	go rule.receive()
	return rule, nil
}

// 将匹配行数和值发送到this.Value通道，重置value和count
func (this *Rule) send() {
	defer func() {
		if e := recover(); e != nil {
		}
	}()
	if this.Type == "avg" {
		if this.count != 0 {
			this.Value <- this.value / this.count
		} else {
			this.Value <- 0
		}
	} else {
		// 如果采集失败，每次也可以推送0
		this.Value <- this.value
	}

	//重置value和count
	this.value = 0
	this.count = 0
}

// 读取line通道，获取文件最新行，按正则条件匹配，累加匹配数。周期发送累加数
func (this *Rule) receive() {
	defer func() {
		if e := recover(); e != nil {
		}
	}()
	var line *string
	ticker := time.NewTicker(time.Second * time.Duration(this.cycle))
	defer ticker.Stop()
	for {
		select {
		case line = <-this.line:
			this.match(line)
		case <-ticker.C:
			// 周期性将匹配行数和值发送到this.Value通道，重置value和count
			// 即使filepath不存在，这里也会周期推送一个0值，让指标持续有数据
			this.send()
		}
	}
}

// 对新行进行正则匹配，计算指标值
func (this *Rule) match(line *string) {
	if !this.subMatch {
		// 整体匹配
		this.oneMatch = this.regexp.FindString(*line)
		if len(this.oneMatch) == 0 {
			return
		}
		this.value = this.value + 1
		return
	}
	// 子匹配
	this.matches = this.regexp.FindStringSubmatch(*line)
	if len(this.matches) == 0 {
		return
	}
	/*
		if len(this.matches) == 1 {
			this.value = this.value + 1
			return
		}
	*/
	if len(this.matches) > 1 {
		for this.tmpvar.index, this.tmpvar.value = range this.matches[1:] {
			this.tmpvar.key = this.subexpNames[this.tmpvar.index+1]
			if this.tmpvar.key == "count" {
				this.tmpvar.value_f64, this.tmpvar.err = strconv.ParseFloat(this.tmpvar.value, 64)
				if this.tmpvar.err == nil {
					this.value = this.value + this.tmpvar.value_f64
					if this.Type == "avg" {
						this.count = this.count + 1
					}
				}
				return
			}
		}
	}
}

// 停止Rule，停止接收和解析新行
func (this *Rule) Stop() {
	// 会使Monitor.Add函数停止
	close(this.Value)
	// 会使Rule.receive和Rule.send函数停止
	close(this.line)
}
