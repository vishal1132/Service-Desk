package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog"
	"github.com/vishal1132/servicedesk/config"
)

func runServer(cfg config.C, logger zerolog.Logger) error {
	// set up signal catching
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, syscall.SIGTERM, syscall.SIGINT)

	rc := redis.NewClient(config.DefaultRedis(cfg))
	defer rc.Close()

	_, cancel := context.WithCancel(context.Background())

	defer cancel()

	hnd := handler{l: &logger}

	mux := http.NewServeMux()

	mux.HandleFunc("/_ruok", hnd.handleRUOK)
	return nil
}
