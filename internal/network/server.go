package network

import (
	"fmt"
	"log"
	"miniredis/internal/store"
	"net"
)

// Server reprezentuje náš TCP server
type Server struct {
	port string
	db   *store.Store
}

// NewServer je konštruktor, ktorý vytvorí novú inštanciu servera
func NewServer(port string, db *store.Store) *Server {
	return &Server{
		port: port,
		db:   db,
	}
}

// Start spustí TCP listener a začne prijímať pripojenia
func (s *Server) Start() error {
	address := fmt.Sprintf(":%s", s.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("Error starting server on port %s: %w", s.port, err)
	}
	defer listener.Close()

	log.Printf("🔥 BlinkDB is running on port %s. Waiting for connections...", s.port)

	// Nekonečný loop na prijímanie klientov
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Printf("Error accepting connection: %v", err)
			continue
		}

		// Start a new goroutine to handle the connection
		go handleConnection(conn, s.db)
	}
}
