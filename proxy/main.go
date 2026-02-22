package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

type Route struct {
	Name      string    `json:"name"`
	Port      int       `json:"port"`
	Healthy   bool      `json:"healthy"`
	LastCheck time.Time `json:"lastCheck"`
	Requests  int64     `json:"-"` // in-memory only, not persisted
}

type State struct {
	mu     sync.RWMutex
	routes map[string]*Route
	file   string
}

func NewState(file string) *State {
	s := &State{
		routes: make(map[string]*Route),
		file:   file,
	}
	s.load()
	return s
}

func (s *State) load() {
	data, err := os.ReadFile(s.file)
	if err != nil {
		log.Printf("no state file found — starting fresh")
		return
	}
	var routes []*Route
	if err := json.Unmarshal(data, &routes); err != nil {
		log.Printf("warning: failed to parse %s: %v", s.file, err)
		return
	}
	for _, r := range routes {
		s.routes[r.Name] = r
	}
	log.Printf("loaded %d routes from %s", len(s.routes), s.file)
}

func (s *State) persist() {
	routes := make([]*Route, 0, len(s.routes))
	for _, r := range s.routes {
		routes = append(routes, r)
	}
	data, err := json.MarshalIndent(routes, "", "  ")
	if err != nil {
		log.Printf("error: failed to marshal state: %v", err)
		return
	}
	if err := os.WriteFile(s.file, data, 0644); err != nil {
		log.Printf("error: failed to write %s: %v", s.file, err)
	}
}

func (s *State) Register(r *Route) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.routes[r.Name] = r
	s.persist()
}

func (s *State) Deregister(name string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.routes[name]; !ok {
		return false
	}
	delete(s.routes, name)
	s.persist()
	return true
}

func (s *State) GetAll() []*Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	routes := make([]*Route, 0, len(s.routes))
	for _, r := range s.routes {
		routes = append(routes, r)
	}
	return routes
}

func (s *State) FindByName(name string) *Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.routes[name]
}

func (s *State) UpdateHealth(name string, healthy bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if r, ok := s.routes[name]; ok {
		r.Healthy = healthy
		r.LastCheck = time.Now()
	}
	s.persist()
}

func checkHealth(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("localhost:%d", port), 2*time.Second)
	if err != nil {
		return false
	}
	conn.Close()
	return true
}

func runHealthChecker(state *State, interval time.Duration) {
	// Check immediately on startup
	for _, r := range state.GetAll() {
		healthy := checkHealth(r.Port)
		state.UpdateHealth(r.Name, healthy)
		status := "unhealthy"
		if healthy {
			status = "healthy"
		}
		log.Printf("startup health check: %s → %s", r.Name, status)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		for _, r := range state.GetAll() {
			healthy := checkHealth(r.Port)
			state.UpdateHealth(r.Name, healthy)
		}
	}
}

func handleRegister(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var route Route
		if err := json.NewDecoder(r.Body).Decode(&route); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if route.Name == "" || route.Port == 0 {
			http.Error(w, "name and port are required", http.StatusBadRequest)
			return
		}
		state.Register(&route)

		// Immediate health check in background
		go func() {
			healthy := checkHealth(route.Port)
			state.UpdateHealth(route.Name, healthy)
			status := "unhealthy"
			if healthy {
				status = "healthy"
			}
			log.Printf("registered %s on port %d → https://%s.dev.kit (%s)", route.Name, route.Port, route.Name, status)
		}()

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "registered %s\n", route.Name)
	}
}

func handleDeregister(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var body struct {
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid JSON: "+err.Error(), http.StatusBadRequest)
			return
		}
		if body.Name == "" {
			http.Error(w, "name is required", http.StatusBadRequest)
			return
		}
		if !state.Deregister(body.Name) {
			http.Error(w, "not found: "+body.Name, http.StatusNotFound)
			return
		}
		log.Printf("deregistered %s", body.Name)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "deregistered %s\n", body.Name)
	}
}

func handleDashboard(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		routes := state.GetAll()
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		fmt.Fprint(w, `<!DOCTYPE html>
<html>
<head>
  <title>devkit</title>
  <meta http-equiv="refresh" content="30">
  <style>
    body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; max-width: 800px; margin: 40px auto; padding: 0 20px; background: #f8f9fa; color: #333; }
    h1 { color: #1a1a1a; }
    table { width: 100%; border-collapse: collapse; background: white; border-radius: 8px; overflow: hidden; box-shadow: 0 1px 3px rgba(0,0,0,0.1); }
    th, td { padding: 12px 16px; text-align: left; border-bottom: 1px solid #eee; }
    th { background: #f1f3f4; font-weight: 600; }
    a { color: #1a73e8; text-decoration: none; }
    a:hover { text-decoration: underline; }
    .healthy { color: #34a853; }
    .unhealthy { color: #ea4335; }
    .empty { text-align: center; padding: 40px; color: #666; }
    .req-count { font-variant-numeric: tabular-nums; }
  </style>
</head>
<body>
  <h1>devkit</h1>
`)
		if len(routes) == 0 {
			fmt.Fprint(w, `  <div class="empty">No tools registered.</div>`)
		} else {
			fmt.Fprint(w, `  <table>
    <tr><th>Tool</th><th>Port</th><th>URL</th><th>Status</th><th>Requests</th><th>Last Checked</th></tr>
`)
			for _, route := range routes {
				status := `<span class="unhealthy">&#10008;</span>`
				if route.Healthy {
					status = `<span class="healthy">&#10004;</span>`
				}
				lastCheck := "never"
				if !route.LastCheck.IsZero() {
					lastCheck = route.LastCheck.Format("15:04:05")
				}
				reqs := atomic.LoadInt64(&route.Requests)
				link := fmt.Sprintf("https://%s.dev.kit", route.Name)
				fmt.Fprintf(w, "    <tr><td>%s</td><td>%d</td><td><a href=\"%s\">%s</a></td><td>%s</td><td class=\"req-count\">%d</td><td>%s</td></tr>\n",
					route.Name, route.Port, link, link, status, reqs, lastCheck)
			}
			fmt.Fprint(w, "  </table>\n")
		}

		fmt.Fprint(w, `
  <h2>API</h2>
  <p><strong>Register:</strong></p>
  <pre>curl -X POST https://home.dev.kit/proxy/register \
  -d '{"name":"myapp", "port":8080}'</pre>
  <p><strong>Deregister:</strong></p>
  <pre>curl -X POST https://home.dev.kit/proxy/deregister \
  -d '{"name":"myapp"}'</pre>
  <p><small><strong>name</strong> and <strong>port</strong> are required. The tool becomes available at <code>https://&lt;name&gt;.dev.kit</code>.</small></p>
`)

		fmt.Fprint(w, "</body>\n</html>\n")
	}
}

func reverseProxy(route *Route) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt64(&route.Requests, 1)
		target, err := url.Parse(fmt.Sprintf("http://localhost:%d", route.Port))
		if err != nil {
			http.Error(w, "bad upstream", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			req.Header.Set("X-Forwarded-Proto", "https")
			req.Header.Set("X-Forwarded-Host", r.Host)
		}
		proxy.ServeHTTP(w, r)
	}
}

func stripPort(host string) string {
	if i := strings.LastIndex(host, ":"); i != -1 {
		return host[:i]
	}
	return host
}

const devkitSuffix = ".dev.kit"
const homeDomain = "home.dev.kit"

func rootHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		host := stripPort(r.Host)

		if host == homeDomain {
			switch r.URL.Path {
			case "/proxy/register":
				handleRegister(state)(w, r)
			case "/proxy/deregister":
				handleDeregister(state)(w, r)
			default:
				handleDashboard(state)(w, r)
			}
			return
		}

		if !strings.HasSuffix(host, devkitSuffix) {
			http.NotFound(w, r)
			return
		}

		subdomain := strings.TrimSuffix(host, devkitSuffix)
		route := state.FindByName(subdomain)
		if route == nil {
			http.NotFound(w, r)
			return
		}

		reverseProxy(route)(w, r)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	tlsPort := envOr("TLS_PORT", "7443")
	certFile := envOr("TLS_CERT", "cert/_wildcard.dev.kit-cert.pem")
	keyFile := envOr("TLS_KEY", "cert/_wildcard.dev.kit-key.pem")
	stateFile := envOr("STATE_FILE", "state.json")
	intervalStr := envOr("HEALTH_CHECK_INTERVAL", "5m")

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("invalid HEALTH_CHECK_INTERVAL: %v", err)
	}

	state := NewState(stateFile)

	go runHealthChecker(state, interval)

	mux := http.NewServeMux()
	mux.HandleFunc("/", rootHandler(state))

	srv := &http.Server{
		Addr:    ":" + tlsPort,
		Handler: mux,
	}

	// Graceful shutdown on SIGTERM/SIGINT
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigCh
		log.Printf("received %s — shutting down", sig)
		state.mu.RLock()
		state.persist()
		state.mu.RUnlock()
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		srv.Shutdown(ctx)
	}()

	log.Printf("devkit proxy listening on https://home.dev.kit (:%s)", tlsPort)
	if err := srv.ListenAndServeTLS(certFile, keyFile); err != http.ErrServerClosed {
		log.Fatalf("server error: %v", err)
	}
	log.Printf("proxy stopped")
}
