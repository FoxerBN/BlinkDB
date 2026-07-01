package network

import (
	"blinkdb/internal/store"
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type Options struct {
	MaxClients               int
	MaxValueBytes            int
	GlobalRateLimitPerSecond int
	IPRateLimitPerSecond     int
	ReadTimeout              time.Duration
	WriteTimeout             time.Duration
	IdleTimeout              time.Duration
	ShutdownTimeout          time.Duration
}

// * Server is the main network layer of BlinkDB. It listens for TCP connections and spawns a goroutine for each client.
type Server struct {
	port          string
	db            *store.Store
	options       Options
	activeClients atomic.Int64
	rateLimiter   *rateLimiter
	listener      net.Listener
	mu            sync.Mutex
	activeConns   map[net.Conn]struct{}
	wg            sync.WaitGroup
	shuttingDown  bool
}

//* NewServer wires the database and runtime limits into the network layer.
func NewServer(port string, db *store.Store, options Options) *Server {
	return &Server{
		port:        port,
		db:          db,
		options:     options,
		rateLimiter: newRateLimiter(options.GlobalRateLimitPerSecond, options.IPRateLimitPerSecond),
		activeConns: make(map[net.Conn]struct{}),
	}
}

//* Start listens for TCP connections and spawns a handler goroutine per client.
func (s *Server) Start() error {
	address := fmt.Sprintf(":%s", s.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("error starting server on port %s: %w", s.port, err)
	}
	defer listener.Close()
	s.mu.Lock()
	if s.shuttingDown {
		s.mu.Unlock()
		return nil
	}
	s.listener = listener
	s.mu.Unlock()

	log.Printf("event=server_start port=%s", s.port)
	log.Printf("event=config max_clients=%d", s.options.MaxClients)
	log.Printf("event=config max_value_bytes=%d", s.options.MaxValueBytes)
	log.Printf("event=config ip_rate_per_second=%d", s.options.IPRateLimitPerSecond)

	for {
		//* Accept a new connection. This is a blocking call, so the server will wait here until a client connects.
		conn, err := listener.Accept()
		if err != nil {
			if s.isShuttingDown() || errors.Is(err, net.ErrClosed) {
				return nil
			}
			log.Printf("event=accept_error error=%q", err)
			continue
		}

		//* Try to reserve a client slot. If the maximum number of clients is reached, reject the connection.
		if !s.tryAddClient() {
			_, _ = conn.Write([]byte("-ERR max clients reached\n"))
			log.Printf("event=client_rejected reason=max_clients addr=%s active_clients=%d max_clients=%d",
				conn.RemoteAddr().String(),
				s.activeClientCount(),
				s.options.MaxClients,
			)
			_ = conn.Close()
			continue
		}
		if !s.addConnection(conn) {
			s.removeClient()
			_ = conn.Close()
			continue
		}

		go func() {
			defer s.wg.Done()
			defer s.removeConnection(conn)
			s.handleConnection(conn)
		}()
	}
}

//* Shutdown stops accepting connections and waits for active ones to finish.
func (s *Server) Shutdown() {
	s.mu.Lock()
	if s.shuttingDown {
		s.mu.Unlock()
		return
	}
	s.shuttingDown = true
	listener := s.listener
	s.mu.Unlock()

	log.Printf("event=shutdown_start")

	if listener != nil {
		_ = listener.Close()
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	timeout := s.options.ShutdownTimeout
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	select {
	case <-done:
	case <-time.After(timeout):
		s.closeActiveConnections()
		<-done
	}

	log.Printf("event=shutdown_done")
}

//* Addr returns the listen address (nil before bind) so tests can find the port.
func (s *Server) Addr() net.Addr {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener == nil {
		return nil
	}
	return s.listener.Addr()
}

//* isShuttingDown reports whether Shutdown has been called.
func (s *Server) isShuttingDown() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.shuttingDown
}

//* addConnection registers an active connection unless the server is shutting down.
func (s *Server) addConnection(conn net.Conn) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.shuttingDown {
		return false
	}
	s.activeConns[conn] = struct{}{}
	s.wg.Add(1)
	return true
}

//* removeConnection removes a connection from the active connections map.
func (s *Server) removeConnection(conn net.Conn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeConns, conn)
}

//* closeActiveConnections force-closes any connections still open at shutdown.
func (s *Server) closeActiveConnections() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for conn := range s.activeConns {
		_ = conn.Close()
	}
}

//* tryAddClient reserves a client slot, returning false when MaxClients is reached.
func (s *Server) tryAddClient() bool {
	if s.options.MaxClients <= 0 {
		s.activeClients.Add(1)
		return true
	}

	//* Keep the active client counter below MaxClients even when many clients connect at the same time.
	for {
		current := s.activeClients.Load()
		if current >= int64(s.options.MaxClients) {
			return false
		}
		if s.activeClients.CompareAndSwap(current, current+1) {
			return true
		}
	}
}

//* removeClient releases the slot reserved by tryAddClient.
func (s *Server) removeClient() {
	s.activeClients.Add(-1)
}

//* activeClientCount returns the current number of connected clients.
func (s *Server) activeClientCount() int64 {
	return s.activeClients.Load()
}

//* clientIP extracts the IP part from RemoteAddr for per-IP rate limiting.
func clientIP(conn net.Conn) string {
	addr := conn.RemoteAddr().String()
	if strings.HasPrefix(addr, "[") {
		end := strings.Index(addr, "]")
		if end > 0 {
			return addr[1:end]
		}
	}

	ip, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return ip
}

// ipBucketTTL is how long a per-IP bucket may sit unused before it is eligible
// for cleanup. cleanupInterval is how often allow() sweeps the map at most.
const (
	ipBucketTTL     = 10 * time.Minute
	cleanupInterval = time.Minute
)

// * rateLimiter tracks command counts for the whole server and for each IP.
type rateLimiter struct {
	mu     sync.Mutex
	global rateBucket
	// perIP grows by one entry per distinct client IP. Stale entries are expired
	// in cleanupLocked so a flood of one-off IPs cannot grow this map forever.
	perIP       map[string]*rateBucket
	globalPS    int
	ipPS        int
	lastCleanup time.Time
}

// * rateBucket stores the count for one fixed one-second window.
type rateBucket struct {
	window   time.Time
	count    int
	lastSeen time.Time
}

//* newRateLimiter creates disabled buckets when limits are <= 0.
func newRateLimiter(globalPS, ipPS int) *rateLimiter {
	return &rateLimiter{
		perIP:    make(map[string]*rateBucket),
		globalPS: globalPS,
		ipPS:     ipPS,
	}
}

//* allow returns true when the command can run now under the rate limits.
func (r *rateLimiter) allow(ip string) bool {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

	r.cleanupLocked(now)

	//* Check the global bucket first. If the global limit is exceeded, reject the command.
	if r.globalPS > 0 && !allowBucket(&r.global, r.globalPS, now) {
		return false
	}

	if r.ipPS <= 0 {
		return true
	}

	bucket := r.perIP[ip]
	if bucket == nil {
		bucket = &rateBucket{}
		r.perIP[ip] = bucket
	}

	bucket.lastSeen = now
	return allowBucket(bucket, r.ipPS, now)
}

//* cleanupLocked expires per-IP buckets unused beyond ipBucketTTL (assumes r.mu held).
func (r *rateLimiter) cleanupLocked(now time.Time) {
	if now.Sub(r.lastCleanup) < cleanupInterval {
		return
	}
	r.lastCleanup = now

	for ip, bucket := range r.perIP {
		if now.Sub(bucket.lastSeen) >= ipBucketTTL {
			delete(r.perIP, ip)
		}
	}
}

//* allowBucket resets the bucket every second and rejects calls over the limit.
func allowBucket(bucket *rateBucket, limit int, now time.Time) bool {
	if bucket.window.IsZero() || now.Sub(bucket.window) >= time.Second {
		bucket.window = now
		bucket.count = 0
	}

	if bucket.count >= limit {
		return false
	}

	bucket.count++
	return true
}

//* checkMemory returns an error if the value exceeds the configured maximum size.
func (s *Server) checkMemory(value []byte) error {
	if s.options.MaxValueBytes > 0 && len(value) > s.options.MaxValueBytes {
		return fmt.Errorf("value exceeds max size of %d bytes", s.options.MaxValueBytes)
	}
	return nil
}
