package g

import (
	"log"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/open-falcon/agent/funcs/logmonitor"
	"github.com/open-falcon/common/model"
	"github.com/toolkits/slice"
)

var Root string

func InitRootDir() {
	var err error
	Root, err = os.Getwd()
	if err != nil {
		log.Fatalln("getwd fail:", err)
	}
}

var LocalIp string

func InitLocalIp() {
	if Config().Heartbeat.Enabled {
		conn, err := net.DialTimeout("tcp", Config().Heartbeat.Addr, time.Second*10)
		if err != nil {
			log.Println("get local addr failed !")
		} else {
			LocalIp = strings.Split(conn.LocalAddr().String(), ":")[0]
			conn.Close()
		}
	} else {
		log.Println("hearbeat is not enabled, can't get localip")
	}
}

var (
	// 保存 HbsClient
	HbsClient *SingleConnRpcClient
)

// 初始化Hbs客户端
func InitRpcClients() {
	if Config().Heartbeat.Enabled {
		HbsClient = &SingleConnRpcClient{
			RpcServer: Config().Heartbeat.Addr,
			Timeout:   time.Duration(Config().Heartbeat.Timeout) * time.Millisecond,
		}
	}
}

// 发送监测数据到transfer
func SendToTransfer(metrics []*model.MetricValue) {
	if len(metrics) == 0 {
		return
	}

	debug := Config().Debug

	if debug {
		log.Printf("=> <Total=%d> %v\n", len(metrics), metrics[0])
	}

	var resp model.TransferResponse
	SendMetrics(metrics, &resp)

	if debug {
		log.Println("<=", &resp)
	}
}

var (
	reportUrls     map[string]string
	reportUrlsLock = new(sync.RWMutex)
)

// 获取url监测指标
func ReportUrls() map[string]string {
	reportUrlsLock.RLock()
	defer reportUrlsLock.RUnlock()
	return reportUrls
}

// 设置url监测指标
func SetReportUrls(urls map[string]string) {
	reportUrlsLock.RLock()
	defer reportUrlsLock.RUnlock()
	reportUrls = urls
}

var (
	reportPorts     []int64
	reportPortsLock = new(sync.RWMutex)
)

func ReportPorts() []int64 {
	reportPortsLock.RLock()
	defer reportPortsLock.RUnlock()
	return reportPorts
}

func SetReportPorts(ports []int64) {
	reportPortsLock.Lock()
	defer reportPortsLock.Unlock()
	reportPorts = ports
}

var (
	duPaths     []string
	duPathsLock = new(sync.RWMutex)
)

func DuPaths() []string {
	duPathsLock.RLock()
	defer duPathsLock.RUnlock()
	return duPaths
}

func SetDuPaths(paths []string) {
	duPathsLock.Lock()
	defer duPathsLock.Unlock()
	duPaths = paths
}

var (
	// tags => {1=>name, 2=>cmdline}
	// e.g. 'name=falcon-agent'=>{1=>falcon-agent}
	// e.g. 'cmdline=xx'=>{2=>xx}
	reportProcs     map[string]map[int]string
	reportProcsLock = new(sync.RWMutex)
)

func ReportProcs() map[string]map[int]string {
	reportProcsLock.RLock()
	defer reportProcsLock.RUnlock()
	return reportProcs
}

func SetReportProcs(procs map[string]map[int]string) {
	reportProcsLock.Lock()
	defer reportProcsLock.Unlock()
	reportProcs = procs
}

var (
	ips     []string
	ipsLock = new(sync.Mutex)
)

func TrustableIps() []string {
	ipsLock.Lock()
	defer ipsLock.Unlock()
	return ips
}

func SetTrustableIps(ipStr string) {
	arr := strings.Split(ipStr, ",")
	ipsLock.Lock()
	defer ipsLock.Unlock()
	ips = arr
}

func IsTrustable(remoteAddr string) bool {
	ip := remoteAddr
	idx := strings.LastIndex(remoteAddr, ":")
	if idx > 0 {
		ip = remoteAddr[0:idx]
	}

	if ip == "127.0.0.1" {
		return true
	}

	return slice.ContainsString(TrustableIps(), ip)
}

var (
	// 从tags中解析filepath和keywords，保存到logs，logs保存多个logmonitor，key为tags字符串。logs最后保存到全局变量reportLogs
	// tags: filepath=/opt/deploy/tiantian/log/tiantian.log,keywords=\[W\]
	// 保存了所有log监控指标，外层key为tags字符串，内层key取值为1和2，分别对应filepath和keywords的值
	reportLogs     map[string]map[int]string
	reportLogsLock = new(sync.RWMutex)
)

func ReportLogs() map[string]map[int]string {
	reportLogsLock.RLock()
	defer reportLogsLock.RUnlock()
	return reportLogs
}

// 保存所有log.monitor监控项
func SetReportLogs(logs map[string]map[int]string) {
	reportLogsLock.Lock()
	defer reportLogsLock.Unlock()
	reportLogs = logs
}

var (
	// newKey := fmt.Sprintf("%s::::::%d", filepath, timestamp)
	// 全局变量，保存了所有monitor 外层key为filepath::::::timestamp，timestamp为最后一次监控时的时间戳，value为对应Monitor
	// logTimeMap在每次内部调用LogMetrics时，都会更新所有key的时间戳，如果key是时间戳长时间没更新，说明此monitor可能被删除了，可以在内存中删除
	// CheckLogMonitor函数就是周期检查过期log.monitor的函数
	// logTimeMap只有一级key，直接代表filepath，没有keywords级别的key。
	logTimeMap     = make(map[string]*logmonitor.Monitor)
	logTimeMapLock = new(sync.RWMutex)
)

func GetLogTimeMap() map[string]*logmonitor.Monitor {
	logTimeMapLock.RLock()
	defer logTimeMapLock.RUnlock()
	return logTimeMap
}

func SetLogTimeMap(key string, monitor *logmonitor.Monitor) {
	logTimeMapLock.Lock()
	defer logTimeMapLock.Unlock()
	logTimeMap[key] = monitor
}

// logTimeMap中删除指定监控
func UpdateLogTimeMap(key string) {
	logTimeMapLock.Lock()
	defer logTimeMapLock.Unlock()
	delete(logTimeMap, key)
}
