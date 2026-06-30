package network

import (
	"blinkdb/internal/store"
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
}

// * Server is the main network layer of BlinkDB. It listens for TCP connections and spawns a goroutine for each client.
type Server struct {
	port          string
	db            *store.Store
	options       Options
	activeClients atomic.Int64
	rateLimiter   *rateLimiter
}

// * NewServer wires the database and runtime limits into the network layer.
func NewServer(port string, db *store.Store, options Options) *Server {
	return &Server{
		port:        port,
		db:          db,
		options:     options,
		rateLimiter: newRateLimiter(options.GlobalRateLimitPerSecond, options.IPRateLimitPerSecond),
	}
}

// * Start begins listening for TCP connections and handles them until the process is terminated. It returns an error if the server cannot start.
func (s *Server) Start() error {
	address := fmt.Sprintf(":%s", s.port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return fmt.Errorf("error starting server on port %s: %w", s.port, err)
	}
	defer listener.Close()

	log.Printf("event=server_start port=%s", s.port)
	log.Printf("event=config max_clients=%d\n max_value_bytes=%d ip_rate_per_second=%d",
		s.options.MaxClients,
		s.options.MaxValueBytes,
		s.options.IPRateLimitPerSecond,
	)

	for {
		//* Accept a new connection. This is a blocking call, so the server will wait here until a client connects.
		conn, err := listener.Accept()
		if err != nil {
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

		go s.handleConnection(conn)
	}
}

// * tryAddClient attempts to increment the active client count. It returns true if successful, or false if the maximum number of clients has been reached.
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

// * removeClient releases the slot reserved by tryAddClient.
func (s *Server) removeClient() {
	s.activeClients.Add(-1)
}

// * activeClientCount is used by STATUS and logs.
func (s *Server) activeClientCount() int64 {
	return s.activeClients.Load()
}

// * clientIP extracts the IP part from RemoteAddr for per-IP rate limiting.
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

// * rateLimiter tracks command counts for the whole server and for each IP.
type rateLimiter struct {
	mu       sync.Mutex
	global   rateBucket
	perIP    map[string]*rateBucket
	globalPS int
	ipPS     int
}

//* rateBucket stores the count for one fixed one-second window.
type rateBucket struct {
	window time.Time
	count  int
}

//* newRateLimiter creates disabled buckets when limits are <= 0.
func newRateLimiter(globalPS, ipPS int) *rateLimiter {
	return &rateLimiter{
		perIP:    make(map[string]*rateBucket),
		globalPS: globalPS,
		ipPS:     ipPS,
	}
}

//* allow returns true when the command can run now.
func (r *rateLimiter) allow(ip string) bool {
	now := time.Now()

	r.mu.Lock()
	defer r.mu.Unlock()

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

	return allowBucket(bucket, r.ipPS, now)
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
