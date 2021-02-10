package main

import (
	lb "dfs/loadbalancer"
	server "dfs/server"
	"flag"
	"fmt"
	"runtime/debug"
)

func main() {

	// print panic stack
	defer func() {
		if err := recover(); err != nil {
			debug.PrintStack()
		}
	}()

	// parse param
	var listenAddr, backendsAddr string
	flag.Bool("?|h", false, "./dfs -l :80 -b http://localhost:2020,http://127.0.0.1:2021")
	flag.StringVar(&listenAddr, "l", ":80", "listen address")
	flag.StringVar(&backendsAddr, "b", "http://localhost:2020,http://127.0.0.1:2021", "backends address")
	flag.Parse()
	parsed := flag.Parsed()
	if parsed != true {
		flag.Usage()
		return
	}
	fmt.Println("listen:[", listenAddr, "] backends:[", backendsAddr, "]")

	// start BackendServer
	go server.Start(backendsAddr)

	// start Loadbalancer
	lb.Start(listenAddr, backendsAddr)

	fmt.Println("dfs exit!")
}
