package plugins

// 保存每个插件文件信息
type Plugin struct {
	FilePath string
	MTime    int64 //修改时间
	Cycle    int   //执行周期
}

var (
	Plugins              = make(map[string]*Plugin)
	PluginsWithScheduler = make(map[string]*PluginScheduler)
)

// 删除不使用的或过期的插件
func DelNoUsePlugins(newPlugins map[string]*Plugin) {
	for currKey, currPlugin := range Plugins {
		newPlugin, ok := newPlugins[currKey]
		if !ok || currPlugin.MTime != newPlugin.MTime {
			deletePlugin(currKey)
		}
	}
}

// 增加新增或更新的插件
func AddNewPlugins(newPlugins map[string]*Plugin) {
	for fpath, newPlugin := range newPlugins {
		if _, ok := Plugins[fpath]; ok && newPlugin.MTime == Plugins[fpath].MTime {
			continue
		}

		Plugins[fpath] = newPlugin
		sch := NewPluginScheduler(newPlugin)
		PluginsWithScheduler[fpath] = sch
		sch.Schedule()
	}
}

// 停止调度并删除所有插件信息
func ClearAllPlugins() {
	for k := range Plugins {
		deletePlugin(k)
	}
}

// 停止调度并删除插件信息
func deletePlugin(key string) {
	v, ok := PluginsWithScheduler[key]
	if ok {
		v.Stop()
		delete(PluginsWithScheduler, key)
	}
	delete(Plugins, key)
}
