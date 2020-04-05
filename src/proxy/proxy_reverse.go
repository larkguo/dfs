package proxy

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

type ReverseProxy struct { // inherit  Handler.ServeHTTP
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// proxy
	trueServer := "http://127.0.0.1:2020"
	url, err := url.Parse(trueServer)
	if err != nil {
		log.Println(err)
		return
	}
	proxy := httputil.NewSingleHostReverseProxy(url)
	proxy.ServeHTTP(w, r)

	fmt.Println("proxy end!")
}

/*
func main() {
	http.HandleFunc("/login", helloHandler)
	log.Fatal(http.ListenAndServe(":2002", nil))
}
*/
