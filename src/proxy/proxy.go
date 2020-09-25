package proxy

import (
	db "dfs/db"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

type HttpProxy struct { // inherit  Handler.ServeHTTP
}

func (p *HttpProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 1.build proxyRequest
	// http请求Header转发
	proxyReq := new(http.Request)
	proxyReq = r
	// http请求增加X-Forwarded-For Header
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		prior, ok := proxyReq.Header["X-Forwarded-For"]
		if ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		proxyReq.Header.Set("X-Forwarded-For", clientIP)
	}

	// 2.transfer request
	// http请求转发和响应接收事务
	resp, err := http.DefaultTransport.RoundTrip(proxyReq)
	if err != nil {
		fmt.Println("RoundTrip:", err)
		http.Error(w, err.Error(), http.StatusServiceUnavailable)
		return
	}
	defer resp.Body.Close()

	// 3.response
	// http响应StatusCode转发
	w.WriteHeader(resp.StatusCode)
	// http响应Header转发
	for key, value := range resp.Header {
		for _, v := range value {
			w.Header().Add(key, v)
		}
	}
	// http响应Body转发
	io.Copy(w, resp.Body) // stream copy

	// 4.调用下一个Handler.ServeHTTP处理: Metadata元数据更新
	if resp.StatusCode == http.StatusOK {
		switch r.Method {
		case http.MethodPut, http.MethodPost, http.MethodDelete, http.MethodHead:
			dbclient := &db.DbClient{}
			dbclient.ServeHTTP(w, r)
		}
	}
}
