package main

import (
	"log"
	"miniredis/internal/network"
	"miniredis/internal/store"
)

func main() {
	port := "6379"
	db := store.NewStore()

	srv := network.NewServer(port, db)

	if err := srv.Start(); err != nil {
		log.Fatalf("Kritická chyba servera: %v", err)
	}
}
