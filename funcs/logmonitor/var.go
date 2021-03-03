/*
 *  Copyright (c) 2015 KingSoft.com, Inc. All Rights Reserved
 *  @file: var.go
 *  @brief:
 *  @author: suxiaolin(suxiaolin@kingsoft.com)
 *  @date: 2015/11/27 20:21:57
 *  @version: 0.0.1
 *  @history:
 */
package logmonitor

import (
	"sync"
	"syscall"
	"time"
)

const (
	FILE_INIT = iota
	FILE_CREATE
	FILE_WAIT_CREATE
	FILE_DELETE
	FILE_MODIFY
	FILE_WAIT_MODIFY
	FILE_TRUNCATE
	FILE_NORMAL
)

const (
	FILE_WATCHER_FLAG = syscall.IN_MODIFY
	DIR_WATCHER_FLAG  = syscall.IN_MOVED_TO | syscall.IN_MOVED_FROM | syscall.IN_CREATE |
		syscall.IN_MOVE_SELF | syscall.IN_DELETE_SELF | syscall.IN_DELETE
)

const (
	WAIT_DIR_CREATE_INTERVAL_TIME     = time.Duration(1) * time.Second /* 1s */
	WAIT_FILE_CREATE_INTERVAL_TIME    = time.Duration(1) * time.Second /* 1s */
	REMOVE_FILE_WATCHER_INTERVAL_TIME = 500000                         /* 500ms */
)

const (
	LINE_BUFFER_SIZE = 1024
)

var (
	// 全局变量，没有使用
	monitorMap     = make(map[string]*Monitor)
	monitorMapLock = new(sync.RWMutex)
	// tags: filepath=/opt/deploy/tiantian/log/tiantian.log,keywords=\[W\]
	// 全局变量，保存了所有log monitor，外层key为filepath，内层key为keywords，value为对应Monitor
	// 可以配置多个log监控，path可以相同但keywords不同
	// logFile里记录了正在运行的monitor
	// logFile里的记录项也需要刷新，比如添加用户新增的，删除用户删除的。
	// logFile更新有两个地方，第一处是agent周期调用LogMetrics获取日志监控结果，这里会使用从hbs同步来的最新reportLogs，
	// reportLogs代表最新的所有日志监控项。在这里可以将用户新添加或更新的监控项添加到logFile中，但不会删除过期的
	// 第二处是在CheckLogMonitor函数中，依据logTimeMap来检查过期的监控项，然后从logFile中删除对应的监控项
	// logFile有两级key，即对于同一file，不同的keywords，其指向的Monitor是同一个
	logFile     = make(map[string]map[string]*Monitor)
	logFileLock = new(sync.RWMutex)
)

func setMonitorMap(v string, m *Monitor) {
	monitorMapLock.RLock()
	defer monitorMapLock.RUnlock()
	monitorMap[v] = m
}

// 获取logFile中的记录
func GetLogFile() map[string]map[string]*Monitor {
	logFileLock.RLock()
	defer logFileLock.RUnlock()
	return logFile
}

// 添加或更新logFile中的记录
func SetLogFile(k string, v string, m *Monitor) {
	logFileLock.Lock()
	defer logFileLock.Unlock()
	if value, existed := logFile[k]; existed {
		if _, existed := value[v]; !existed {
			value[v] = m
			logFile[k] = value
		}
	} else {
		mm := make(map[string]*Monitor)
		mm[v] = m
		logFile[k] = mm
	}

}

// 删除在logFile中的记录
func UpdateLogFile(key string) {
	logFileLock.Lock()
	defer logFileLock.Unlock()
	delete(logFile, key)
}
