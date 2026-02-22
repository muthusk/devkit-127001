package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"
)

type Route struct {
	Name        string    `json:"name"`
	Port        int       `json:"port"`
	Path        string    `json:"path,omitempty"`
	StripPrefix *bool     `json:"stripPrefix,omitempty"`
	Healthy     bool      `json:"healthy"`
	LastCheck   time.Time `json:"lastCheck"`
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
	if r.Path != "" && !strings.HasPrefix(r.Path, "/") {
		r.Path = "/" + r.Path
	}
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

func (s *State) FindByPath(reqPath string) *Route {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var best *Route
	for _, r := range s.routes {
		if r.Path == "" {
			continue
		}
		if strings.HasPrefix(reqPath, r.Path) {
			if best == nil || len(r.Path) > len(best.Path) {
				best = r
			}
		}
	}
	return best
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
			pathInfo := "(no proxy path)"
			if route.Path != "" {
				pathInfo = "at " + route.Path
			}
			log.Printf("registered %s on port %d %s (%s)", route.Name, route.Port, pathInfo, status)
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

func handleDashboard(state *State, proxyPort string) http.HandlerFunc {
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
  </style>
</head>
<body>
  <h1>devkit</h1>
`)
		if len(routes) == 0 {
			fmt.Fprint(w, `  <div class="empty">No tools registered.</div>`)
		} else {
			fmt.Fprint(w, `  <table>
    <tr><th>Tool</th><th>Port</th><th>Proxy Path</th><th>Status</th><th>Last Checked</th></tr>
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
				var link, pathCol string
				if route.Path != "" {
					link = fmt.Sprintf("http://localhost:%s%s", proxyPort, route.Path)
					pathCol = route.Path
				} else {
					link = fmt.Sprintf("http://localhost:%d", route.Port)
					pathCol = "—"
				}
				fmt.Fprintf(w, "    <tr><td><a href=\"%s\">%s</a></td><td>%d</td><td>%s</td><td>%s</td><td>%s</td></tr>\n",
					link, route.Name, route.Port, pathCol, status, lastCheck)
			}
			fmt.Fprint(w, "  </table>\n")
		}

		fmt.Fprintf(w, `
  <h2>API</h2>
  <p><strong>Register</strong> (with proxy routing):</p>
  <pre>curl -X POST http://localhost:%s/proxy/register \
  -d '{"name":"myapp", "port":8080, "path":"/myapp"}'</pre>
  <p><strong>Register</strong> (health monitor only, no proxy routing):</p>
  <pre>curl -X POST http://localhost:%s/proxy/register \
  -d '{"name":"postgres", "port":5432}'</pre>
  <p><strong>Deregister:</strong></p>
  <pre>curl -X POST http://localhost:%s/proxy/deregister \
  -d '{"name":"myapp"}'</pre>
  <p><small><strong>name</strong> and <strong>port</strong> are required. <strong>path</strong> is optional — when provided, the proxy routes <code>localhost:%s/&lt;path&gt;/*</code> to <code>localhost:&lt;port&gt;/*</code>.</small></p>
`, proxyPort, proxyPort, proxyPort, proxyPort)

		fmt.Fprint(w, "</body>\n</html>\n")
	}
}

func proxyHandler(state *State) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		route := state.FindByPath(r.URL.Path)
		if route == nil {
			http.NotFound(w, r)
			return
		}
		target, err := url.Parse(fmt.Sprintf("http://localhost:%d", route.Port))
		if err != nil {
			http.Error(w, "bad upstream", http.StatusBadGateway)
			return
		}
		proxy := httputil.NewSingleHostReverseProxy(target)
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			// Strip prefix by default; skip when StripPrefix is explicitly false
			if route.StripPrefix == nil || *route.StripPrefix {
				req.URL.Path = strings.TrimPrefix(req.URL.Path, route.Path)
				if req.URL.Path == "" {
					req.URL.Path = "/"
				}
			}
			req.URL.RawPath = ""
		}
		proxy.ServeHTTP(w, r)
	}
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	port := envOr("PROXY_PORT", "7001")
	stateFile := envOr("STATE_FILE", "state.json")
	intervalStr := envOr("HEALTH_CHECK_INTERVAL", "5m")

	interval, err := time.ParseDuration(intervalStr)
	if err != nil {
		log.Fatalf("invalid HEALTH_CHECK_INTERVAL: %v", err)
	}

	state := NewState(stateFile)

	go runHealthChecker(state, interval)

	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/register", handleRegister(state))
	mux.HandleFunc("/proxy/deregister", handleDeregister(state))

	// Catch-all handler: dashboard for root, proxy for everything else
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handleDashboard(state, port)(w, r)
			return
		}
		proxyHandler(state)(w, r)
	})

	addr := ":" + port
	log.Printf("devkit proxy listening on http://localhost:%s", port)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
