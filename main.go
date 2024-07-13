package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/fcgi"
	"os"

	"github.com/go-chi/chi/v5"
)

type serveType int8

type serverConfig struct {
	ServerType serveType
	HttpPort   string
}

const (
	serveTypeHttp serveType = iota
	serveTypeFcgi
)

func loadServerConfig() *serverConfig {
	return &serverConfig{
		ServerType: serveTypeHttp,
		HttpPort:   "9000",
	}
}

func run() error {
	config := loadServerConfig()

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello World")
	})

	switch config.ServerType {
	case serveTypeHttp:
		if config.HttpPort == "" {
			return fmt.Errorf("http port not defined")
		}

		port := config.HttpPort
		log.Println("Server listening on port ", port)
		if err := http.ListenAndServe(":"+port, r); err != nil {
			return fmt.Errorf("cannot run server: %v", err)
		}
	case serveTypeFcgi:
		if err := fcgi.Serve(nil, r); err != nil {
			return fmt.Errorf("cannot run server: %v", err)
		}
	default:
		panic("cannot run any server type")
	}
	return nil
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
