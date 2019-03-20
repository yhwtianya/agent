package http

import (
	"net/http"
	"os"
	"time"

	"github.com/open-falcon/agent/g"
	"github.com/toolkits/file"
)

// 管理命令管理命令
func configAdminRoutes() {
	// 退出
	http.HandleFunc("/exit", func(w http.ResponseWriter, r *http.Request) {
		if g.IsTrustable(r.RemoteAddr) {
			w.Write([]byte("exiting..."))
			go func() {
				time.Sleep(time.Second)
				os.Exit(0)
			}()
		} else {
			w.Write([]byte("no privilege"))
		}
	})

	// 热加载文件
	http.HandleFunc("/config/reload", func(w http.ResponseWriter, r *http.Request) {
		if g.IsTrustable(r.RemoteAddr) {
			g.ParseConfig(g.ConfigFile)
			RenderDataJson(w, g.Config())
		} else {
			w.Write([]byte("no privilege"))
		}
	})

	// workdir
	http.HandleFunc("/workdir", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, file.SelfDir())
	})

	// TrustableIps
	http.HandleFunc("/ips", func(w http.ResponseWriter, r *http.Request) {
		RenderDataJson(w, g.TrustableIps())
	})
}
