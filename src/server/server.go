package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

type Server struct { // inherit  Handler.ServeHTTP
	ListenAddr string
	Path       string
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPut:
		s.objectPut(w, r)
	case http.MethodPost:
		s.objectPut(w, r)
	case http.MethodGet:
		s.objectGet(w, r)
	case http.MethodDelete:
		s.objectDelete(w, r)
	}
}

func (s *Server) objectPut(w http.ResponseWriter, r *http.Request) {

	fullname := s.Path + r.URL.Path
	objectpath := filepath.Dir(fullname)
	err := os.MkdirAll(objectpath, os.ModePerm)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("fullname:", fullname)
	fp, err := os.Create(fullname)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	defer fp.Close()
	io.Copy(fp, r.Body)
}

func (s *Server) objectGet(w http.ResponseWriter, r *http.Request) {
	fullname := s.Path + r.URL.Path
	fmt.Println("ObjectGet:", fullname)
	fp, err := os.Open(fullname)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
	defer fp.Close()
	io.Copy(w, fp)
}

func (s *Server) objectDelete(w http.ResponseWriter, r *http.Request) {
	fullname := s.Path + r.URL.Path
	err := os.Remove(fullname)
	if err != nil {
		fmt.Println(err)
		w.WriteHeader(http.StatusNotFound)
		return
	}
}

func Start(backendsAddr string) {

	if backendsAddr == "" {
		fmt.Println("no backend server start!")
		return
	}

	backends := strings.Split(backendsAddr, ",")
	for _, v := range backends {
		go func(addr string) {
			url, err := url.Parse(addr)
			if err == nil {
				server := Server{ListenAddr: addr, Path: "./" + url.Scheme + "_" + url.Hostname() + "_" + url.Port()}
				err := os.MkdirAll(server.Path, os.ModePerm)
				if err != nil {
					fmt.Println(err)
					return
				}

				s := &http.Server{
					Addr:    url.Host,
					Handler: &server,
				}
				fmt.Println("Server:", server.ListenAddr, " Path:", server.Path)
				fmt.Println(s.ListenAndServe())
			}
		}(v)
	}

	select {} // block
}
