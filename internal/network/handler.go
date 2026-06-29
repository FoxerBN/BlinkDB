package network

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"strings"
	"time"
)

// handleConnection owns one client connection from connect to disconnect.
// It reads line-based commands, executes them, and writes line-based responses.
func (s *Server) handleConnection(conn net.Conn) {
	clientAddr := conn.RemoteAddr().String()
	ip := clientIP(conn)
	log.Printf("event=client_connected addr=%s ip=%s active_clients=%d", clientAddr, ip, s.activeClientCount())
	defer func() {
		s.removeClient()
		_ = conn.Close()
		log.Printf("event=client_disconnected addr=%s ip=%s active_clients=%d", clientAddr, ip, s.activeClientCount())
	}()

	if !s.writeResponse(conn, s.statusResponse()) {
		return
	}

	// Scanner reads the TCP stream line by line, so every command must end with
	// a newline. That keeps the protocol easy to test with nc or telnet.
	scanner := bufio.NewScanner(conn)
	maxCommandBytes := s.maxCommandBytes()
	// Scanner has a small default token limit; raise it to match the configured
	// max value size so large SET commands are rejected deliberately, not by accident.
	scanner.Buffer(make([]byte, 0, 1024), maxCommandBytes)

	for {
		s.setReadDeadline(conn)
		if !scanner.Scan() {
			break
		}

		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			continue
		}

		// Rate limiting happens before parsing so even invalid command spam is
		// counted and limited.
		if !s.rateLimiter.allow(ip) {
			log.Printf("event=command_rejected reason=rate_limit addr=%s ip=%s", clientAddr, ip)
			if !s.writeResponse(conn, "-ERR rate limit exceeded\n") {
				return
			}
			continue
		}

		// ParseCommand validates the command name and argument count. The handler
		// can then focus only on executing known, valid commands.
		command, err := ParseCommand(text)
		if err != nil {
			if !s.writeResponse(conn, fmt.Sprintf("-ERR %v\n", err)) {
				return
			}
			continue
		}

		response, shouldClose := s.executeCommand(command)
		if !s.writeResponse(conn, response) {
			return
		}
		if shouldClose {
			return
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("event=read_error addr=%s ip=%s error=%q", clientAddr, ip, err)
	}
}

// executeCommand is the bridge between parsed protocol commands and store
// operations. The bool return tells the caller whether the connection should close.
func (s *Server) executeCommand(command *Command) (string, bool) {
	switch command.Name {
	case "PING":
		return "+PONG\n", false

	case "STATUS":
		return s.statusResponse(), false

	case "SET":
		key := command.Args[0]
		value := command.Args[1]
		if s.options.MaxValueBytes > 0 && len(value) > s.options.MaxValueBytes {
			return "-ERR value too large\n", false
		}

		s.db.Set(key, value)
		return "+OK\n", false

	case "GET":
		key := command.Args[0]
		value, exist := s.db.Get(key)
		if !exist {
			return "-ERR Key not found\n", false
		}
		return fmt.Sprintf("+%s\n", value), false

	case "DELETE":
		key := command.Args[0]
		deleted := s.db.Delete(key)
		if !deleted {
			return "-ERR Key not found\n", false
		}
		return "+OK\n", false

	case "QUIT", "EXIT":
		return "+BYE\n", true

	default:
		return fmt.Sprintf("-ERR Unknown command: %s\n", command.Name), false
	}
}

// statusResponse is shared by the initial greeting and the STATUS command.
func (s *Server) statusResponse() string {
	return fmt.Sprintf("+STATUS OK clients=%d keys=%d\n", s.activeClientCount(), s.db.Count())
}

// writeResponse applies the configured write timeout before writing to the socket.
func (s *Server) writeResponse(conn net.Conn, response string) bool {
	s.setWriteDeadline(conn)
	if _, err := conn.Write([]byte(response)); err != nil {
		log.Printf("event=write_error addr=%s error=%q", conn.RemoteAddr().String(), err)
		return false
	}
	return true
}

// setReadDeadline protects the server from clients that connect but stop sending data.
func (s *Server) setReadDeadline(conn net.Conn) {
	timeout := s.options.ReadTimeout
	if timeout <= 0 {
		timeout = s.options.IdleTimeout
	}
	setDeadline(conn, timeout, conn.SetReadDeadline)
}

// setWriteDeadline protects the server from clients that stop reading responses.
func (s *Server) setWriteDeadline(conn net.Conn) {
	setDeadline(conn, s.options.WriteTimeout, conn.SetWriteDeadline)
}

// setDeadline centralizes deadline handling for read and write paths.
func setDeadline(conn net.Conn, timeout time.Duration, setter func(time.Time) error) {
	if timeout <= 0 {
		return
	}
	if err := setter(time.Now().Add(timeout)); err != nil {
		log.Printf("event=deadline_error addr=%s error=%q", conn.RemoteAddr().String(), err)
	}
}

// maxCommandBytes gives Scanner enough room for "SET key value" plus protocol
// overhead while still enforcing a configured upper bound.
func (s *Server) maxCommandBytes() int {
	if s.options.MaxValueBytes <= 0 {
		return 1024 * 1024
	}
	return s.options.MaxValueBytes + 4096
}
