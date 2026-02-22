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

func testServer(state *State) *httptest.Server {
	return httptest.NewServer(rootHandler(state))
}

// postJSONWithHost sends a POST to the test server with a custom Host header.
func postJSONWithHost(ts *httptest.Server, path, host, body string) (*http.Response, string) {
	req, err := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	if err != nil {
		return nil, ""
	}
	req.Host = host
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

// getWithHost sends a GET to the test server with a custom Host header.
func getWithHost(ts *httptest.Server, path, host string) (*http.Response, string) {
	req, err := http.NewRequest(http.MethodGet, ts.URL+path, nil)
	if err != nil {
		return nil, ""
	}
	req.Host = host
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, ""
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp, string(b)
}

func TestRegister(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	resp, body := postJSONWithHost(ts, "/proxy/register", "home.dev.kit", `{"name":"web","port":8080}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	routes := state.GetAll()
	if len(routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(routes))
	}
	if routes[0].Name != "web" || routes[0].Port != 8080 {
		t.Fatalf("unexpected route: %+v", routes[0])
	}
}

func TestRegisterValidation(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	// Missing name
	resp, _ := postJSONWithHost(ts, "/proxy/register", "home.dev.kit", `{"port":8080}`)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing name, got %d", resp.StatusCode)
	}

	// Missing port
	resp, _ = postJSONWithHost(ts, "/proxy/register", "home.dev.kit", `{"name":"test"}`)
	if resp.StatusCode != 400 {
		t.Fatalf("expected 400 for missing port, got %d", resp.StatusCode)
	}
}

func TestDeregister(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	postJSONWithHost(ts, "/proxy/register", "home.dev.kit", `{"name":"web","port":8080}`)
	time.Sleep(100 * time.Millisecond) // let background health check goroutine finish

	resp, _ := postJSONWithHost(ts, "/proxy/deregister", "home.dev.kit", `{"name":"web"}`)
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
	ts := testServer(state)
	defer ts.Close()

	resp, _ := postJSONWithHost(ts, "/proxy/deregister", "home.dev.kit", `{"name":"nope"}`)
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestDashboardEmpty(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	resp, html := getWithHost(ts, "/", "home.dev.kit")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	if !strings.Contains(html, "No tools registered") {
		t.Fatal("expected empty state message")
	}
	if !strings.Contains(html, "/proxy/register") {
		t.Fatal("expected API docs on dashboard")
	}
}

func TestDashboardWithRoutes(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	postJSONWithHost(ts, "/proxy/register", "home.dev.kit", `{"name":"myapp","port":9090}`)
	// Let background health check goroutine finish
	time.Sleep(100 * time.Millisecond)

	_, html := getWithHost(ts, "/", "home.dev.kit")

	if !strings.Contains(html, "myapp") {
		t.Fatal("expected tool name on dashboard")
	}
	if !strings.Contains(html, "9090") {
		t.Fatal("expected port on dashboard")
	}
	if !strings.Contains(html, "https://myapp.dev.kit") {
		t.Fatal("expected subdomain URL on dashboard")
	}
}

func TestDashboardAPIDocumentation(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	_, html := getWithHost(ts, "/", "home.dev.kit")

	if !strings.Contains(html, "https://home.dev.kit/proxy/register") {
		t.Fatal("expected register endpoint in API docs")
	}
	if !strings.Contains(html, "https://home.dev.kit/proxy/deregister") {
		t.Fatal("expected deregister endpoint in API docs")
	}
	if !strings.Contains(html, `https://&lt;name&gt;.dev.kit`) {
		t.Fatal("expected subdomain pattern in API docs")
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

func TestSubdomainProxy(t *testing.T) {
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
	state.Register(&Route{Name: "app", Port: upstreamPort})

	ts := testServer(state)
	defer ts.Close()

	// Request through the proxy with subdomain host
	resp, body := getWithHost(ts, "/hello", "app.dev.kit")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}

	// Upstream should see /hello (path passed through unchanged)
	expected := "upstream:/hello"
	if body != expected {
		t.Fatalf("expected %q, got %q", expected, body)
	}
}

func TestSubdomainProxyRootPath(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "upstream:%s", r.URL.Path)
	}))
	defer upstream.Close()

	upstreamPort := 0
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &upstreamPort)

	state := newTestState(t)
	state.Register(&Route{Name: "app", Port: upstreamPort})

	ts := testServer(state)
	defer ts.Close()

	_, body := getWithHost(ts, "/", "app.dev.kit")
	expected := "upstream:/"
	if body != expected {
		t.Fatalf("expected %q, got %q", expected, body)
	}
}

func TestProxyNoRouteMatch(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	resp, _ := getWithHost(ts, "/anything", "nonexistent.dev.kit")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404, got %d", resp.StatusCode)
	}
}

func TestBadHostHeader(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	// Host that doesn't match *.dev.kit pattern
	resp, _ := getWithHost(ts, "/", "example.com")
	if resp.StatusCode != 404 {
		t.Fatalf("expected 404 for non-devkit host, got %d", resp.StatusCode)
	}
}

func TestHomeDomainDashboard(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	// home.dev.kit serves dashboard
	resp, html := getWithHost(ts, "/", "home.dev.kit")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(html, "<title>devkit</title>") {
		t.Fatal("expected dashboard HTML")
	}
}

func TestHomeDomainRegister(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	// Registration works on home.dev.kit
	resp, body := postJSONWithHost(ts, "/proxy/register", "home.dev.kit", `{"name":"svc","port":3000}`)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d: %s", resp.StatusCode, body)
	}

	if state.FindByName("svc") == nil {
		t.Fatal("expected route to be registered")
	}
	time.Sleep(100 * time.Millisecond) // let background health check goroutine finish
}

func TestHomeDomainWithPort(t *testing.T) {
	state := newTestState(t)
	ts := testServer(state)
	defer ts.Close()

	// home.dev.kit:7443 should still match (port stripped)
	resp, html := getWithHost(ts, "/", "home.dev.kit:7443")
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(html, "<title>devkit</title>") {
		t.Fatal("expected dashboard HTML with port in host")
	}
}

func TestRequestCounter(t *testing.T) {
	// Start an upstream HTTP server
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, "ok")
	}))
	defer upstream.Close()

	upstreamPort := 0
	fmt.Sscanf(upstream.URL, "http://127.0.0.1:%d", &upstreamPort)

	state := newTestState(t)
	state.Register(&Route{Name: "counter-test", Port: upstreamPort})

	ts := testServer(state)
	defer ts.Close()

	// Send 3 requests through the proxy
	for i := 0; i < 3; i++ {
		getWithHost(ts, "/", "counter-test.dev.kit")
	}

	route := state.FindByName("counter-test")
	if route == nil {
		t.Fatal("expected route to exist")
	}
	if route.Requests != 3 {
		t.Fatalf("expected 3 requests, got %d", route.Requests)
	}

	// Dashboard should show the count
	_, html := getWithHost(ts, "/", "home.dev.kit")
	if !strings.Contains(html, ">3<") {
		t.Fatal("expected request count of 3 on dashboard")
	}
}

func TestStatePersistence(t *testing.T) {
	stateFile := tempStateFile(t)

	// Create state and register a route
	s1 := NewState(stateFile)
	s1.Register(&Route{Name: "persist-test", Port: 3000})

	// Create a new state from the same file
	s2 := NewState(stateFile)
	routes := s2.GetAll()

	if len(routes) != 1 {
		t.Fatalf("expected 1 persisted route, got %d", len(routes))
	}
	if routes[0].Name != "persist-test" || routes[0].Port != 3000 {
		t.Fatalf("unexpected persisted route: %+v", routes[0])
	}
}

func TestStateFileFormat(t *testing.T) {
	stateFile := tempStateFile(t)
	state := NewState(stateFile)
	state.Register(&Route{Name: "test", Port: 8080})

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

	// path and stripPrefix should not be present
	if _, ok := routes[0]["path"]; ok {
		t.Fatal("state file should not contain path field")
	}
	if _, ok := routes[0]["stripPrefix"]; ok {
		t.Fatal("state file should not contain stripPrefix field")
	}
}
