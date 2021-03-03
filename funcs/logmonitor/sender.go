package logmonitor

type Sender interface {
	Init() error
	Send(map[string]*RuleData)
	GetMetricValue() map[string]*RuleData
	Stop()
}
