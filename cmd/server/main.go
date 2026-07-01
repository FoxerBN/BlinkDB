package main

import (
	"blinkdb/internal/config"
	"blinkdb/internal/network"
	"blinkdb/internal/store"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
)

//* main wires config, store, and network server together and blocks until shutdown.
func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("blinkdb ")

	cfg := config.Load(".env")
	if cfg.MemoryMB > 0 {
		debug.SetMemoryLimit(int64(cfg.MemoryMB) * 1024 * 1024)
	}

	db := store.NewStore()
	srv := network.NewServer(cfg.Port, db, network.Options{
		MaxClients:               cfg.MaxClients,
		MaxValueBytes:            cfg.MaxValueBytes,
		GlobalRateLimitPerSecond: cfg.GlobalRateLimitPerSecond,
		IPRateLimitPerSecond:     cfg.IPRateLimitPerSecond,
		ReadTimeout:              cfg.ReadTimeout,
		WriteTimeout:             cfg.WriteTimeout,
		IdleTimeout:              cfg.IdleTimeout,
		ShutdownTimeout:          cfg.ShutdownTimeout,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signalCh)

	select {
	case err := <-errCh:
		if err != nil {
			log.Fatalf("Critical error: %v", err)
		}
	case sig := <-signalCh:
		log.Printf("event=shutdown_signal signal=%s", sig)
		srv.Shutdown()
		if err := <-errCh; err != nil {
			log.Fatalf("Critical error: %v", err)
		}
	}
}
