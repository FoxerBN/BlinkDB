package network

import (
	"bufio"
	"fmt"
	"log"
	"miniredis/internal/store"
	"net"
	"strings"
)

func handleConnection(conn net.Conn, db *store.Store) {
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	log.Printf("🟢 Client connected: %s", clientAddr)
	defer log.Printf("🔴 Client disconnected: %s", clientAddr)

	if _, err := fmt.Fprintf(conn, "+STATUS OK keys=%d\n", db.Count()); err != nil {
		log.Printf("Error during status check for %s: %v", clientAddr, err)
		return
	}

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		command, err := ParseCommand(text)
		if err != nil {
			response := fmt.Sprintf("-ERR %v\n", err)

			if _, writeErr := conn.Write([]byte(response)); writeErr != nil {
				log.Printf("Error during error response for %s: %v", clientAddr, writeErr)
				return
			}
			continue
		}

		shouldClose := false
		var response string

		switch command.Name {
		case "PING":
			response = "+PONG\n"

		case "STATUS":
			response = fmt.Sprintf("+STATUS OK keys=%d\n", db.Count())

		case "SET":
			key := command.Args[0]
			value := command.Args[1]
			db.Set(key, value)
			response = "+OK\n"

		case "GET":
			key := command.Args[0]
			value, exist := db.Get(key)
			if !exist {
				response = "-ERR Key not found\n"
			} else {
				response = fmt.Sprintf("+%s\n", value)
			}

		case "DELETE":
			key := command.Args[0]
			deleted := db.Delete(key)
			if !deleted {
				response = "-ERR Key not found\n"
			} else {
				response = "+OK\n"
			}

		case "QUIT", "EXIT":
			response = "+BYE\n"
			shouldClose = true

		default:
			response = fmt.Sprintf("-ERR Unknown command: %s\n", text)
		}

		if _, err := conn.Write([]byte(response)); err != nil {
			log.Printf(
				"Error during response for %s: %v",
				clientAddr,
				err,
			)
			return
		}

		if shouldClose {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error during reading from client %s: %v", clientAddr, err)
	}
}
