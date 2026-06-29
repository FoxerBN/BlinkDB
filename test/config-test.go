package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// TestConfig contains values shared by all manual load tests in this folder.
type TestConfig struct {
	Name    string
	Host    string
	Port    string
	Users   int
	Hold    time.Duration
	Timeout time.Duration
}

var config = TestConfig{
	Name:    "MiniRedis stress test",
	Host:    "localhost",
	Port:    "6379",
	Users:   20000,
	Hold:    2 * time.Second,
	Timeout: 5 * time.Second,
}

type testFunc func(TestConfig, int) error

var tests = map[string]testFunc{}

func main() {
	testName := flag.String("test", "stress", "test to run")
	flag.Parse()

	fn, ok := tests[*testName]
	if !ok {
		fmt.Printf("unknown test %q\n", *testName)
		fmt.Printf("available tests: %v\n", testNames())
		os.Exit(1)
	}

	config.Name = fmt.Sprintf("MiniRedis %s test", *testName)
	runTest(config, fn)
}

// registerTest makes a scenario runnable with: go run . -test <name>.
func registerTest(name string, fn testFunc) {
	tests[name] = fn
}

// Address returns the TCP address used by net.Dial.
func (c TestConfig) Address() string {
	return net.JoinHostPort(c.Host, c.Port)
}

// runTest wraps one concrete scenario with shared start/end logs and counters.
func runTest(cfg TestConfig, fn testFunc) {
	var processed atomic.Int64
	var failed atomic.Int64
	var firstError string
	var firstErrorOnce sync.Once
	var wg sync.WaitGroup

	start := time.Now()
	fmt.Println(cfg.Name)
	fmt.Printf("start address=%s users=%d hold=%s timeout=%s\n",
		cfg.Address(),
		cfg.Users,
		cfg.Hold,
		cfg.Timeout,
	)

	for clientID := range cfg.Users {
		wg.Go(func() {
			if err := fn(cfg, clientID); err != nil {
				firstErrorOnce.Do(func() {
					firstError = err.Error()
				})
				failed.Add(1)
				return
			}

			processed.Add(1)
		})
	}

	wg.Wait()

	elapsed := time.Since(start).Round(time.Millisecond)
	fmt.Printf("done processed=%d failed=%d elapsed=%s\n", processed.Load(), failed.Load(), elapsed)
	if firstError != "" {
		fmt.Printf("first_error=%q\n", firstError)
	}
	fmt.Printf("rate processed_per_second=%.2f\n", float64(processed.Load())/elapsed.Seconds())
}

func testNames() []string {
	names := make([]string, 0, len(tests))
	for name := range tests {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
