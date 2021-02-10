package loadbalancer

import (
	db "dfs/db"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type ObjectItemResp struct {
	Name    string `json:"name"`
	Backend string `json:"backend"`
	Hash    string `json:"hash"`
}
type Backend struct {
	URL   *url.URL
	Alive uint
}
type LoadBalancer struct { // inherit  Handler.ServeHTTP
	backends []*Backend
}

var instance *LoadBalancer
var once sync.Once

func NewSingleton() *LoadBalancer {
	once.Do(func() {
		instance = &LoadBalancer{}
	})
	return instance
}

func (b *Backend) SetAlive(alive uint) {
	b.Alive = alive
	db.UpdateBackendStatus(b.URL.Scheme, b.URL.Host, b.Alive)
}

func (b *Backend) IsAlive() (alive uint) {
	alive = b.Alive
	return
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

func (l *LoadBalancer) AddBackend(backend *Backend) {
	l.backends = append(l.backends, backend)
	fmt.Println("AddBackend:", backend.URL, backend.Alive)
}

func (l *LoadBalancer) HealthCheck() {
	t := time.NewTicker(30 * time.Second)
	for {
		select {
		case <-t.C:
			for _, b := range l.backends {
				alive := isBackendAlive(b.URL)
				b.SetAlive(alive)
			}
		}
	}
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
	backendStr := db.GetNextBackend() // BackendServer(smallest magic)
	if backendStr != "" {
		backend, err := url.Parse(backendStr)
		if err == nil {
			// -> BackendServer
			r.URL.Scheme = backend.Scheme
			r.URL.Host = backend.Host
			fmt.Println("backend:", backend.Scheme, backend.Host)
			resp, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				fmt.Println("RoundTrip:", err)
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer resp.Body.Close()

			// -> Client
			w.WriteHeader(resp.StatusCode)
			for key, value := range resp.Header {
				for _, v := range value {
					w.Header().Add(key, v)
				}
			}
			io.Copy(w, resp.Body) // stream copy

			// -> MetadataDB
			if resp.StatusCode == http.StatusOK {
				dbclient := &db.DbClient{}
				dbclient.ServeHTTP(w, r)
			}
			return
		}
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func lbGetObject(w http.ResponseWriter, r *http.Request) {
	backendStr := db.GetBackendByObject(r)
	if backendStr != "" {
		backend, err := url.Parse(backendStr)
		if err == nil {
			//  -> BackendServer
			r.URL.Scheme = backend.Scheme
			r.URL.Host = backend.Host
			fmt.Println("backend:", backend.Scheme, backend.Host)
			resp, err := http.DefaultTransport.RoundTrip(r)
			if err != nil {
				fmt.Println("RoundTrip:", err)
				http.Error(w, err.Error(), http.StatusServiceUnavailable)
				return
			}
			defer resp.Body.Close()

			// -> Client
			w.WriteHeader(resp.StatusCode)
			for key, value := range resp.Header {
				for _, v := range value {
					w.Header().Add(key, v)
				}
			}
			io.Copy(w, resp.Body) // stream copy

			return
		}
	}
	http.Error(w, "Service not available", http.StatusServiceUnavailable)
}

func lbDeleteObject(w http.ResponseWriter, r *http.Request) {
	backendStr := db.GetBackendByObject(r)
	url, err := url.Parse(backendStr)
	if err == nil {
		// -> BackendServer
		r.URL.Scheme = url.Scheme
		r.URL.Host = url.Host
		fmt.Println("backend:", url.Scheme, url.Host)
		resp, err := http.DefaultTransport.RoundTrip(r)
		if err != nil {
			fmt.Println("RoundTrip:", err)
			http.Error(w, err.Error(), http.StatusServiceUnavailable)
			return
		}
		defer resp.Body.Close()

		// ->Client
		w.WriteHeader(resp.StatusCode)
		for key, value := range resp.Header {
			for _, v := range value {
				w.Header().Add(key, v)
			}
		}
		io.Copy(w, resp.Body) // stream copy

		// -> MetadataDB
		if resp.StatusCode == http.StatusOK {
			dbclient := &db.DbClient{}
			dbclient.ServeHTTP(w, r)
		}
	}
}

func lbGetObjectInfo(w http.ResponseWriter, r *http.Request) {
	// MetadataDB
	dbclient := &db.DbClient{}
	dbclient.ServeHTTP(w, r)
}

func Start(listenAddr, backendsAddr string) {
	var count int

	instance := NewSingleton()

	// BackendServer
	if backendsAddr != "" {
		backends := strings.Split(backendsAddr, ",")
		for _, v := range backends {
			url, err := url.Parse(v)
			if err == nil {
				count += 1
				instance.AddBackend(&Backend{URL: url, Alive: 1})
				db.AddBackend(url.Scheme, url.Host, 1, 0, 0, 0, 0, 1)
			}
		}
	}

	// BackendServer from MetadataDB
	if count == 0 {
		fmt.Println("Get backends from db...")
		backendItems := db.GetAllBackends()
		for _, v := range backendItems {
			url, err := url.Parse(v.Backend)
			if err == nil {
				instance.AddBackend(&Backend{URL: url, Alive: v.Alive})
			}
		}
	}

	go instance.HealthCheck()

	l := &http.Server{
		Addr:    listenAddr,
		Handler: &LoadBalancer{},
	}
	fmt.Println(l.ListenAndServe())
}
