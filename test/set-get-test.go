package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	tokenKey1   = "id_token1"
	tokenValue1 = "546843554ewfwe5f4354644%#$%22f165efr3e"
	tokenKey2   = "id_token2"
	tokenValue2 = "546843sdfsdwfwe5s@!$&**#$%22f1sfr3e"
)

var prepareSetGetOnce sync.Once
var prepareSetGetErr error

//* init registers the set-get scenario at startup.
func init() {
	registerTest("set-get", runSetGetClient)
}

//* runSetGetClient seeds data once, then GETs this client's key and checks the value.
func runSetGetClient(cfg TestConfig, clientID int) error {
	// Store both test values once before clients start reading them.
	prepareSetGetOnce.Do(func() {
		prepareSetGetErr = prepareSetGetData(cfg)
	})
	if prepareSetGetErr != nil {
		return prepareSetGetErr
	}

	key, expectedValue := setGetTarget(clientID, cfg.Users)

	// Open one TCP connection for this simulated user.
	conn, err := net.DialTimeout("tcp", cfg.Address(), cfg.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// Read the server greeting that BlinkDB sends after connect.
	if _, err := readLine(conn, reader, cfg.Timeout); err != nil {
		return err
	}

	// Ask for one of the prepared values.
	if err := writeLine(conn, cfg.Timeout, "GET %s", key); err != nil {
		return err
	}

	// The response must contain the exact value for this client's key.
	response, err := readLine(conn, reader, cfg.Timeout)
	if err != nil {
		return err
	}
	if response != "+"+expectedValue {
		return fmt.Errorf("unexpected GET response for %s: %s", key, response)
	}

	// Keep the connection open so this also tests concurrent active clients.
	time.Sleep(cfg.Hold)

	return writeLine(conn, cfg.Timeout, "QUIT")
}

//* prepareSetGetData stores both test tokens once before clients read them.
func prepareSetGetData(cfg TestConfig) error {
	conn, err := net.DialTimeout("tcp", cfg.Address(), cfg.Timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	if _, err := readLine(conn, reader, cfg.Timeout); err != nil {
		return err
	}
	if err := setToken(conn, reader, cfg, tokenKey1, tokenValue1); err != nil {
		return err
	}
	if err := setToken(conn, reader, cfg, tokenKey2, tokenValue2); err != nil {
		return err
	}

	return writeLine(conn, cfg.Timeout, "QUIT")
}

//* setToken sends one SET and verifies the +OK response.
func setToken(conn net.Conn, reader *bufio.Reader, cfg TestConfig, key string, value string) error {
	if err := writeLine(conn, cfg.Timeout, "SET %s %s", key, value); err != nil {
		return err
	}

	response, err := readLine(conn, reader, cfg.Timeout)
	if err != nil {
		return err
	}
	if response != "+OK" {
		return fmt.Errorf("unexpected SET response for %s: %s", key, response)
	}

	return nil
}

//* setGetTarget splits clients across the two tokens by client ID.
func setGetTarget(clientID int, users int) (string, string) {
	if clientID < users/2 {
		return tokenKey1, tokenValue1
	}
	return tokenKey2, tokenValue2
}

//* readLine reads one trimmed response line under a read deadline.
func readLine(conn net.Conn, reader *bufio.Reader, timeout time.Duration) (string, error) {
	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(line), nil
}

//* writeLine writes one formatted command line under a write deadline.
func writeLine(conn net.Conn, timeout time.Duration, format string, args ...any) error {
	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	_, err := fmt.Fprintf(conn, format+"\n", args...)
	return err
}
