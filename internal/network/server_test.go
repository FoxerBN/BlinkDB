package network

import (
	"bufio"
	"net"
	"testing"
	"time"

	"blinkdb/internal/store"
)

func TestShutdownStopsServerAndClosesIdleClients(t *testing.T) {
	srv := NewServer("0", store.NewStore(), Options{
		ShutdownTimeout: 25 * time.Millisecond,
	})

	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.Start()
	}()

	listener := waitForListener(t, srv, errCh)
	conn, err := net.DialTimeout("tcp", listener.Addr().String(), time.Second)
	if err != nil {
		t.Fatalf("DialTimeout() error = %v", err)
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)
	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := reader.ReadString('\n'); err != nil {
		t.Fatalf("ReadString() greeting error = %v", err)
	}

	shutdownDone := make(chan struct{})
	go func() {
		srv.Shutdown()
		close(shutdownDone)
	}()

	select {
	case <-shutdownDone:
	case <-time.After(time.Second):
		t.Fatal("Shutdown() did not return")
	}

	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("Start() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("Start() did not return after Shutdown()")
	}

	_ = conn.SetReadDeadline(time.Now().Add(time.Second))
	if _, err := reader.ReadString('\n'); err == nil {
		t.Fatal("client connection is still open after Shutdown()")
	}
}

func waitForListener(t *testing.T, srv *Server, errCh <-chan error) net.Listener {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(time.Millisecond)
	defer ticker.Stop()

	for {
		srv.mu.Lock()
		listener := srv.listener
		srv.mu.Unlock()
		if listener != nil {
			return listener
		}

		select {
		case err := <-errCh:
			t.Fatalf("Start() returned before listener was ready: %v", err)
		case <-deadline:
			t.Fatal("listener was not ready")
		case <-ticker.C:
		}
	}
}
