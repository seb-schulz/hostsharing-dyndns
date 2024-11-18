package main

import (
	"log/slog"
	"net/http"
	"os"
	timepk "time"

	"github.com/go-chi/httplog/v2"
)

type loggerConfig struct {
	Enabled bool
	Level   slog.Level
	File    string
}

func loggerMiddleware(config loggerConfig) func(next http.Handler) http.Handler {
	opt := httplog.Options{
		JSON:            true,
		LogLevel:        config.Level,
		Concise:         true,
		RequestHeaders:  true,
		ResponseHeaders: true,
		TimeFieldFormat: timepk.RFC3339,
		Tags: map[string]string{
			"version": "latest",
		},
		QuietDownRoutes: []string{"/query", "/favicon.ico"},
		QuietDownPeriod: timepk.Minute,
	}

	if config.File != "" {
		f, err := os.OpenFile(config.File, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0640)
		if err != nil {
			panic(err)
		}
		opt.Writer = f
	}
	return httplog.RequestLogger(httplog.NewLogger("hostsharing-dyndns", opt))
}
