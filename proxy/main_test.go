package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func tempStateFile(t *testing.T) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "state-*.json")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	os.Remove(f.Name()) // start clean — NewState handles missing file
	return f.Name()
}

func newTestState(t *testing.T) *State {
	t.Helper()
	return NewState(tempStateFile(t))
}

func testMux(state *State, port string) *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/proxy/register", handleRegister(state))
	mux.HandleFunc("/proxy/deregister", handleDeregister(state))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			handleDashboard(state, port)(w, r)
			return
		}
		proxyHandler(state)(w, r)
	})
	return mux
}

func postJSON(ts *httptest.Server, path string, body string) (*http.Response, string) {
	resp, err := http.Post(ts.URL+path, "application/json", strings.NewReader(body))
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestRegister(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	resp, body := postJSON(ts, "/proxy/register", `{"name":"web","port":8080,"path":"/web"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	routes := state.GetAll()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Name != "web" || routes[0].Port != 8080 || routes[0].Path != "/web" {
		t.Fatalf("unexpected route: %+v", routes[0])
	}
}

func TestRegisterWithoutPath(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	resp, body := postJSON(ts, "/proxy/register", `{"name":"postgres","port":5432}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	routes := state.GetAll()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Path != "" {
		t.Fatalf("expected empty path, got %q", routes[0].Path)
	}
}

func TestRegisterValidation(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	// Missing name
	resp, _ := postJSON(ts, "/proxy/register", `{"port":8080}`)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing name, got %d", resp.StatusCode)
	}

	// Missing port
	resp, _ = postJSON(ts, "/proxy/register", `{"name":"test"}`)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing port, got %d", resp.StatusCode)
	}
}

func TestDeregister(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	postJSON(ts, "/proxy/register", `{"name":"web","port":8080,"path":"/web"}`)

	resp, _ := postJSON(ts, "/proxy/deregister", `{"name":"web"}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	routes := state.GetAll()
	if len(routes) != 0 {
		t.Fatalf("expected 0 routes, got %d", len(routes))
	}
}

func TestDeregisterNotFound(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	resp, _ := postJSON(ts, "/proxy/deregister", `{"name":"nope"}`)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDashboardEmpty(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "No tools registered") {
		t.Fatal("expected empty state message")
	}
	if !strings.Contains(html, "/proxy/register") {
		t.Fatal("expected API docs on dashboard")
	}
}

func TestDashboardWithRoutes(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	postJSON(ts, "/proxy/register", `{"name":"myapp","port":9090,"path":"/myapp"}`)
	// Let background health check goroutine finish
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "myapp") {
		t.Fatal("expected tool name on dashboard")
	}
	if !strings.Contains(html, "9090") {
		t.Fatal("expected port on dashboard")
	}
	if !strings.Contains(html, "/myapp") {
		t.Fatal("expected proxy path on dashboard")
	}
}

func TestDashboardWithPathlessRoute(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	postJSON(ts, "/proxy/register", `{"name":"postgres","port":5432}`)
	time.Sleep(100 * time.Millisecond)

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	// Should link directly to the port, not through proxy
	if !strings.Contains(html, "localhost:5432") {
		t.Fatal("expected direct link to port for pathless route")
	}
	// Path column should show dash
	if !strings.Contains(html, "\xe2\x80\x94") { // em dash
		t.Fatal("expected dash in path column for pathless route")
	}
}

func TestDashboardAPIDocumentation(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	html := string(body)

	if !strings.Contains(html, "/proxy/register") {
		t.Fatal("expected register endpoint in API docs")
	}
	if !strings.Contains(html, "/proxy/deregister") {
		t.Fatal("expected deregister endpoint in API docs")
	}
	if !strings.Contains(html, "health monitor only") {
		t.Fatal("expected pathless registration example in API docs")
	}
}

func TestHealthCheckTCP(t *testing.T) {
	// Start a TCP listener on a random port
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port

	// Healthy when listening
	if !checkHealth(port) {
		t.Fatal("expected healthy when port is listening")
	}

	// Unhealthy after closing
	ln.Close()
	if checkHealth(port) {
		t.Fatal("expected unhealthy when port is closed")
	}
}

func TestProxyRouting(t *testing.T) {
	// Start an upstream HTTP server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "upstream:%s", r.URL.Path)
	}))
	defer upstream.Close()

	// Extract the upstream port
	upstreamPort := 0
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &upstreamPort)
	if upstreamPort == 0 {
		t.Fatalf("could not parse upstream port from %s", upstream.URL)
	}

	state := newTestState(t)
	state.Register(&Route{Name: "app", Port: upstreamPort, Path: "/app"})

	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	// Request through the proxy
	resp, err := http.Get(ts.URL + "/app/hello")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Upstream should see /hello (prefix /app stripped)
	expected := "upstream:/hello"
	if string(body) != expected {
		t.Fatalf("expected %q, got %q", expected, string(body))
	}
}

func TestProxyRoutingNoStripPrefix(t *testing.T) {
	// Start an upstream HTTP server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "upstream:%s", r.URL.Path)
	}))
	defer upstream.Close()

	upstreamPort := 0
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &upstreamPort)
	if upstreamPort == 0 {
		t.Fatalf("could not parse upstream port from %s", upstream.URL)
	}

	state := newTestState(t)
	noStrip := false
	state.Register(&Route{Name: "kc", Port: upstreamPort, Path: "/keycloak", StripPrefix: &noStrip})

	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	// Request through the proxy — path should NOT be stripped
	resp, err := http.Get(ts.URL + "/keycloak/admin/master")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)

	// Upstream should see /keycloak/admin/master (prefix preserved)
	expected := "upstream:/keycloak/admin/master"
	if string(body) != expected {
		t.Fatalf("expected %q, got %q", expected, string(body))
	}
}

func TestProxyNoRouteMatch(t *testing.T) {
	state := newTestState(t)
	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/nonexistent")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestProxySkipsPathlessRoutes(t *testing.T) {
	state := newTestState(t)
	state.Register(&Route{Name: "db", Port: 5432})

	ts := httptest.NewServer(testMux(state, "7001"))
	defer ts.Close()

	// Even though "db" is registered, it has no path — should not proxy
	resp, err := http.Get(ts.URL + "/db")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for pathless route, got %d", resp.StatusCode)
	}
}

func TestStatePersistence(t *testing.T) {
	stateFile := tempStateFile(t)

	// Create state and register a route
	s1 := NewState(stateFile)
	s1.Register(&Route{Name: "persist-test", Port: 3000, Path: "/persist"})

	// Create a new state from the same file
	s2 := NewState(stateFile)
	routes := s2.GetAll()

	if len(routes) != 1 {
		t.Fatalf("expected 1 persisted route, got %d", len(routes))
	}
	if routes[0].Name != "persist-test" || routes[0].Port != 3000 || routes[0].Path != "/persist" {
		t.Fatalf("unexpected persisted route: %+v", routes[0])
	}
}

func TestStateFileFormat(t *testing.T) {
	stateFile := tempStateFile(t)
	state := NewState(stateFile)
	state.Register(&Route{Name: "test", Port: 8080, Path: "/test"})

	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatal(err)
	}

	var routes []map[string]interface{}
	if err := json.Unmarshal(data, &routes); err != nil {
		t.Fatal(err)
	}

	if len(routes) != 1 {
		t.Fatalf("expected 1 route in JSON, got %d", len(routes))
	}

	// healthURL should not be present
	if _, ok := routes[0]["healthURL"]; ok {
		t.Fatal("state file should not contain healthURL field")
	}
}
