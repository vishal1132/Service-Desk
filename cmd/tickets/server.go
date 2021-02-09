package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-redis/redis"
	"github.com/rs/zerolog"
	"github.com/vishal1132/servicedesk/config"
)

func getserviceup() {
	// Get all the tickets whose status is pending and put them back in the array

}

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
	go getserviceup()
	mux.HandleFunc("/_ruok", hnd.handleRUOK)
	mux.HandleFunc("/registercompany", hnd.handleRegisterCompany)
	mux.HandleFunc("/createTicket", hnd.handleCreateTicket)
	mux.HandleFunc("/registerAgents", hnd.handleRegisterAgents)
	socketAddr := fmt.Sprintf("0.0.0.0:%d", cfg.Port)
	logger.Info().
		Str("addr", socketAddr).
		Msg("binding to TCP socket")

	listener, err := net.Listen("tcp", socketAddr)
	if err != nil {
		return fmt.Errorf("failed to open HTTP socket: %w", err)
	}

	defer func() { _ = listener.Close() }()

	// set up the HTTP server
	httpSrvr := &http.Server{
		Handler:     mux,
		ReadTimeout: 20 * time.Second,
		IdleTimeout: 60 * time.Second,
	}

	serveStop, serverShutdown := make(chan struct{}), make(chan struct{})
	var serveErr, shutdownErr error

	// HTTP server parent goroutine
	go func() {
		defer close(serveStop)
		serveErr = httpSrvr.Serve(listener)
	}()

	// signal handling / graceful shutdown goroutine
	go func() {
		defer close(serverShutdown)
		sig := <-signalCh

		logger.Info().
			Str("signal", sig.String()).
			Msg("shutting HTTP server down gracefully")

		cctx, ccancel := context.WithTimeout(context.Background(), 25*time.Second)

		defer ccancel()
		defer cancel()

		if shutdownErr = httpSrvr.Shutdown(cctx); shutdownErr != nil {
			logger.Error().
				Err(shutdownErr).
				Msg("failed to gracefully shut down HTTP server")
		}
	}()

	// wait for it to die
	<-serverShutdown
	<-serveStop

	// log errors for informational purposes
	logger.Info().
		AnErr("serve_err", serveErr).
		AnErr("shutdown_err", shutdownErr).
		Msg("server shut down")
	return nil
}
