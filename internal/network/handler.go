package network

import (
	"bufio"
	"fmt"
	"log"
	"miniredis/internal/store"
	"miniredis/internal/network/protocol"
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

		// ParseCommand ma spravit normalizaciu a validaciu vstupu.
		// Handler by potom mal pracovat uz len s command.Name a command.Args.
		command, err := protocol.ParseCommand(text)
		if err != nil {
			// TODO: tu by bolo lepsie poslat klientovi "-ERR ...\n" a pokracovat
			// na dalsi command cez continue. Return klienta hned odpoji.
			log.Printf("Chyba pri spracovaní príkazu od klienta %s: %v", clientAddr, err)
			return
		}

		var response string

		// command.Name je uz uppercase z ParseCommand.
		// command.Args ostavaju v povodnom tvare, aby si nemenil key/value.
		switch command.Name {
		case "PING":
			response = "+PONG\n"

		case "STATUS":
			response = fmt.Sprintf("+STATUS OK keys=%d\n", db.Count())

		// TODO: ked doplnis store operacie do handlera:
		// case "GET":
		//     // key bude command.Args[0]
		//     // db.Get(key) vrati value a bool, podla toho posli odpoved
		//
		// case "SET":
		//     // key bude command.Args[0], value bude command.Args[1]
		//     // db.Set(key, value), potom posli OK odpoved
		//
		// case "DELETE":
		//     // key bude command.Args[0]
		//     // db.Delete(key) vrati bool, podla toho posli odpoved

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
			// Ak ParseCommand validuje unknown command, toto by sa normalne nemalo stat.
			// Nechaj to ako poistku pre pripad, ze sa validacie neskor zmenia.
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
