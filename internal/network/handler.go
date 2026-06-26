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
	log.Printf("🟢 Nový klient pripojený: %s", clientAddr)
	defer log.Printf("🔴 Klient odpojený: %s", clientAddr)

	if _, err := conn.Write([]byte(fmt.Sprintf("+STATUS OK keys=%d\n", db.Count()))); err != nil {
		log.Printf("Chyba pri odosielaní odpovede klientovi %s: %v", clientAddr, err)
		return
	}

	scanner := bufio.NewScanner(conn)

	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		command := strings.ToUpper(text)

		var response string

		switch command {
		case "PING":
			response = "+PONG\n"

		case "STATUS":
			response = fmt.Sprintf("+STATUS OK keys=%d\n", db.Count())

		case "QUIT", "EXIT":
			response = "+BYE\n"

			if _, err := conn.Write([]byte(response)); err != nil {
				log.Printf(
					"Chyba pri odosielaní odpovede klientovi %s: %v",
					clientAddr,
					err,
				)
			}

			return

		default:
			response = fmt.Sprintf("-ERR Neznámy príkaz: %s\n", text)
		}

		if _, err := conn.Write([]byte(response)); err != nil {
			log.Printf(
				"Chyba pri odosielaní odpovede klientovi %s: %v",
				clientAddr,
				err,
			)
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Chyba pri čítaní od klienta %s: %v", clientAddr, err)
	}
}
