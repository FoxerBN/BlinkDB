package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"time"
)

const stressCommand = "STATUS"

func init() {
	registerTest("stress", runStressClient)
}

func runStressClient(cfg TestConfig, _ int) error {
	// Open one TCP connection to the running BlinkDB server.
	conn, err := net.DialTimeout("tcp", cfg.Address(), cfg.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// BlinkDB sends a STATUS line immediately after a client connects.
	_ = conn.SetReadDeadline(time.Now().Add(cfg.Timeout))
	greeting, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !strings.HasPrefix(greeting, "+STATUS OK") {
		return fmt.Errorf("unexpected greeting: %s", strings.TrimSpace(greeting))
	}

	// Send the command this stress scenario is measuring.
	_ = conn.SetWriteDeadline(time.Now().Add(cfg.Timeout))
	if _, err := fmt.Fprintf(conn, "%s\n", stressCommand); err != nil {
		return err
	}

	// Read and validate the command response. Error responses count as failed.
	_ = conn.SetReadDeadline(time.Now().Add(cfg.Timeout))
	response, err := reader.ReadString('\n')
	if err != nil {
		return err
	}
	if !validStressResponse(response) {
		return fmt.Errorf("unexpected response: %s", strings.TrimSpace(response))
	}

	// Keep the connection open for a while to test many active clients at once.
	time.Sleep(cfg.Hold)

	// Ask the server to close the connection cleanly.
	_ = conn.SetWriteDeadline(time.Now().Add(cfg.Timeout))
	_, err = fmt.Fprint(conn, "QUIT\n")
	return err
}

func validStressResponse(response string) bool {
	response = strings.TrimSpace(response)
	return strings.HasPrefix(response, "+STATUS OK")
}
