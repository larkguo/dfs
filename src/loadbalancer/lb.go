package loadbalancer

import (
	db "dfs/db"
	proxy "dfs/proxy"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Backend struct {
	URL   *url.URL
	Alive uint
}
type LoadBalancer struct { // inherit  Handler.ServeHTTP
	backends []*Backend
}

var g_loadBalancer LoadBalancer

func init() {
	go healthCheck()
}

func (b *Backend) SetAlive(alive uint) {
	b.Alive = alive
	db.UpdateBackendStatus(b.URL.Scheme, b.URL.Host, b.Alive)
}

func (b *Backend) IsAlive() (alive uint) {
	alive = b.Alive
	return
}

func (l *LoadBalancer) AddBackend(backend *Backend) {
	l.backends = append(l.backends, backend)
	fmt.Println("AddBackend:", backend.URL, backend.Alive)
}

func (l *LoadBalancer) HealthCheck() {
	for _, b := range l.backends {
		alive := isBackendAlive(b.URL)
		b.SetAlive(alive)
	}
}

type ObjectItemResp struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	//Size      uint64 `json:"size"`
	Hash string `json:"hash"`
}

func (l *LoadBalancer) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodPut, http.MethodPost:
		lbPutObject(w, r)
	case http.MethodGet:
		lbGetObject(w, r)
	case http.MethodDelete:
		lbDeleteObject(w, r)
		//case http.MethodHead:
		//	lbGetObjectInfo(w, r)
	}
}

func lbPutObject(w http.ResponseWriter, r *http.Request) {
	// 根据策略(magic最小)获取下一个后端处理的地址
	backendStr := db.GetNextBackend()
	if backendStr != "" {
		backend, err := url.Parse(backendStr)
		if err == nil {

			// 设置后续Handler.ServeHTTP处理的URL
			r.URL.Scheme = backend.Scheme
			r.URL.Host = backend.Host
			fmt.Println("backend:", backend.Scheme, backend.Host)

			// 调用下一个Handler.ServeHTTP处理: proxy代理
			p := &proxy.HttpProxy{}
			p.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func lbGetObject(w http.ResponseWriter, r *http.Request) {
	// find backend
	backendStr := db.GetBackendByObject(r)
	if backendStr != "" {
		backend, err := url.Parse(backendStr)
		if err == nil {
			// set backend url
			r.URL.Scheme = backend.Scheme
			r.URL.Host = backend.Host
			fmt.Println("backend:", backend.Scheme, backend.Host)

			// 调用下一个Handler.ServeHTTP处理: proxy代理
			p := &proxy.HttpProxy{}
			p.ServeHTTP(w, r)
			return
		}
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func lbDeleteObject(w http.ResponseWriter, r *http.Request) {
	// find backend
	backendStr := db.GetBackendByObject(r)
	url, err := url.Parse(backendStr)
	if err == nil {
		// set backend url
		r.URL.Scheme = url.Scheme
		r.URL.Host = url.Host
		fmt.Println("backend:", url.Scheme, url.Host)

		// 调用下一个Handler.ServeHTTP处理: proxy代理
		p := &proxy.HttpProxy{}
		p.ServeHTTP(w, r)
	}
}

func lbGetObjectInfo(w http.ResponseWriter, r *http.Request) {
	// 调用下一个Handler.ServeHTTP处理: Metadata元数据获取
	dbclient := &db.DbClient{}
	dbclient.ServeHTTP(w, r)
}

func isBackendAlive(u *url.URL) uint {
	timeout := time.Second
	conn, err := net.DialTimeout("tcp", u.Host, timeout)
	if err != nil {
		fmt.Println("Site unreachable, error: ", err)
		return 0
	}
	_ = conn.Close()
	return 1
}

func healthCheck() {
	t := time.NewTicker(time.Minute)
	for {
		select {
		case <-t.C:
			g_loadBalancer.HealthCheck()
		}
	}
}

func Start(listenAddr, backendsAddr string) {
	var count int

	// backends from command
	if backendsAddr != "" {
		backends := strings.Split(backendsAddr, ",")
		for _, v := range backends {
			url, err := url.Parse(v)
			if err == nil {
				count += 1
				g_loadBalancer.AddBackend(&Backend{URL: url, Alive: 1})
				db.AddBackend(url.Scheme, url.Host, 1, 0, 0, 0, 0, 1)
			}
		}
	}

	// backends from db
	if count == 0 {
		fmt.Println("Get backends from db...")
		backendItems := db.GetAllBackends()
		for _, v := range backendItems {
			url, err := url.Parse(v.Backend)
			if err == nil {
				g_loadBalancer.AddBackend(&Backend{URL: url, Alive: v.Alive})
			}
		}
	}

	l := &http.Server{
		Addr:    listenAddr,
		Handler: &LoadBalancer{},
	}
	fmt.Println(l.ListenAndServe())
}
