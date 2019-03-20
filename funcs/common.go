package funcs

import (
	"strings"

	"github.com/open-falcon/common/model"
)

// 构建Metric实例，没有对endpoint、step、timestamp赋值
func NewMetricValue(metric string, val interface{}, dataType string, tags ...string) *model.MetricValue {
	mv := model.MetricValue{
		Metric: metric,
		Value:  val,
		Type:   dataType,
	}

	size := len(tags)

	if size > 0 {
		mv.Tags = strings.Join(tags, ",")
	}

	return &mv
}

// 构建Gauge类型Metric实例，没有对endpoint、step、timestamp赋值
func GaugeValue(metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValue(metric, val, "GAUGE", tags...)
}

// 构建Counter类型Metric实例，没有对endpoint、step、timestamp赋值
func CounterValue(metric string, val interface{}, tags ...string) *model.MetricValue {
	return NewMetricValue(metric, val, "COUNTER", tags...)
}
