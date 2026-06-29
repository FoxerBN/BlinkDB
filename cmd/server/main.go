package main

import (
	"log"
	"miniredis/internal/config"
	"miniredis/internal/network"
	"miniredis/internal/store"
	"runtime/debug"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lmicroseconds)
	log.SetPrefix("miniredis ")

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
	})

	if err := srv.Start(); err != nil {
		log.Fatalf("Critical error: %v", err)
	}
}
