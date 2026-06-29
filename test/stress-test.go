package main

import (
	"bufio"
	"fmt"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

var (
	address = "localhost:6379"
	users   = 20000
	hold    = 2 * time.Second
	timeout = 5 * time.Second
	command = "STATUS"
)

func main() {
	var processed atomic.Int64
	var failed atomic.Int64
	var wg sync.WaitGroup

	start := time.Now()
	fmt.Println("MiniRedis stress test")
	fmt.Printf("start address=%s users=%d command=%s hold=%s timeout=%s\n", address, users, command, hold, timeout)

	for range users {
		wg.Go(func() {
			if err := runClient(); err != nil {
				failed.Add(1)
				return
			}

			processed.Add(1)
		})
	}

	wg.Wait()

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("done processed=%d failed=%d elapsed=%s\n", processed.Load(), failed.Load(), elapsed)
	fmt.Printf("rate processed_per_second=%.2f\n", float64(processed.Load())/elapsed.Seconds())
}

func runClient() error {
	conn, err := net.DialTimeout("tcp", address, timeout)
	if err != nil {
		return err
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	if _, err := reader.ReadString('\n'); err != nil {
		return err
	}

	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	if _, err := fmt.Fprintf(conn, "%s\n", command); err != nil {
		return err
	}

	_ = conn.SetReadDeadline(time.Now().Add(timeout))
	if _, err := reader.ReadString('\n'); err != nil {
		return err
	}

	time.Sleep(hold)

	_ = conn.SetWriteDeadline(time.Now().Add(timeout))
	_, err = fmt.Fprint(conn, "QUIT\n")
	return err
}
