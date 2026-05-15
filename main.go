package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"

	"github.com/exerayy/yadro-netlog-parser/internal/adapters/db"
	handlers "github.com/exerayy/yadro-netlog-parser/internal/adapters/http"
	"github.com/exerayy/yadro-netlog-parser/internal/config"
	"github.com/exerayy/yadro-netlog-parser/internal/core/parser"
	"github.com/exerayy/yadro-netlog-parser/internal/core/topology"
	httpSwagger "github.com/swaggo/http-swagger"

	_ "github.com/exerayy/yadro-netlog-parser/docs"
)

// @title Yadro Netlog Parser
// @version 1.0
// @description API для работы логами

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "server configuration file")
	flag.Parse()

	cfg := config.MustLoad(configPath)

	address := ":" + cfg.Port

	log := mustMakeLogger(cfg.LogLevel)

	log.Info("starting server")
	log.Debug("debug messages are enabled")

	storage, err := db.New(log, cfg.DBAddress)
	if err != nil {
		log.Error("failed to connect to db", "error", err)
		os.Exit(1)
	}
	err = storage.Migrate()
	if err != nil {
		log.Error("failed to migrate db", "error", err)
		os.Exit(1)
	}
	defer storage.Close()

	serviceParser := parser.New(log, storage, cfg.DataDir)
	serviceTopology := topology.New(log, storage)

	mux := http.NewServeMux()

	mux.Handle("GET /swagger/", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	mux.Handle("GET /api/v1/health", handlers.NewHealthHandler(log))
	mux.Handle("POST /api/v1/parse", handlers.NewParseHandler(log, serviceParser))
	mux.Handle("GET /api/v1/topology/{log_id}", handlers.NewGetTopologyHandler(log, serviceTopology))
	mux.Handle("GET /api/v1/node/{node_id}", handlers.NewGetNodeHandler(log, serviceTopology))
	mux.Handle("GET /api/v1/port/{node_id}", handlers.NewGetNodePortsHandler(log, serviceTopology))
	mux.Handle("GET /api/v1/log/{log_id}", handlers.NewGetLogHandler(log, serviceTopology))

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	server := http.Server{
		Addr:        address,
		ReadTimeout: cfg.Timeout,
		Handler:     mux,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
	}

	go func() {
		<-ctx.Done()
		log.Debug("shutting down server")
		if err := server.Shutdown(context.Background()); err != nil {
			log.Error("erroneous shutdown", "error", err)
		}
	}()

	log.Info("Running HTTP server",
		"address", address,
		"swagger_ui", "http://localhost"+address+"/swagger/index.html",
	)
	if err := server.ListenAndServe(); err != nil {
		if !errors.Is(err, http.ErrServerClosed) {
			log.Error("server closed unexpectedly", "error", err)
			return
		}
	}
}

func mustMakeLogger(logLevel string) *slog.Logger {
	var level slog.Level
	switch logLevel {
	case "DEBUG":
		level = slog.LevelDebug
	case "INFO":
		level = slog.LevelInfo
	case "ERROR":
		level = slog.LevelError
	default:
		panic("unknown log level: " + logLevel)
	}

	handler := slog.NewJSONHandler(
		os.Stdout,
		&slog.HandlerOptions{
			Level:     level,
			AddSource: true,
		})

	return slog.New(handler)
}
