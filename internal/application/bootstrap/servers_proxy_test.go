package bootstrap

import (
	"bufio"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"sql-graph-visualizer/internal/domain/models"
)

// The viz server has a short WriteTimeout; SSE responses proxied through it
// must still be delivered incrementally and survive past that timeout.
func TestAPIProxy_StreamsSSEPastWriteTimeout(t *testing.T) {
	release := make(chan struct{})
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher := w.(http.Flusher)
		_, _ = w.Write([]byte("event: sample\ndata: 1\n\n"))
		flusher.Flush()
		<-release
		_, _ = w.Write([]byte("event: done\ndata: 2\n\n"))
		flusher.Flush()
	}))
	defer backend.Close()

	backendURL, err := url.Parse(backend.URL)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.Handle("/api/", newAPIProxy(backendURL.Port()))
	front := httptest.NewUnstartedServer(mux)
	front.Config.WriteTimeout = 300 * time.Millisecond
	front.Start()
	defer front.Close()

	resp, err := http.Get(front.URL + "/api/performance/benchmarks/x/stream")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	reader := bufio.NewReader(resp.Body)

	// First event arrives immediately, i.e. the proxy flushes instead of buffering.
	first := readEvent(t, reader)
	if !strings.Contains(first, "event: sample") {
		t.Fatalf("first event = %q, want the sample event", first)
	}

	// Hold the stream open longer than the server WriteTimeout, then finish it.
	time.Sleep(600 * time.Millisecond)
	close(release)

	second := readEvent(t, reader)
	if !strings.Contains(second, "event: done") {
		t.Fatalf("second event = %q, want the done event (stream was cut by WriteTimeout?)", second)
	}
}

func TestAPIProxy_BackendDown(t *testing.T) {
	// Port 1 is never listening.
	front := httptest.NewServer(newAPIProxy("1"))
	defer front.Close()

	resp, err := http.Get(front.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", resp.StatusCode)
	}
}

func TestDemoModeEnabled(t *testing.T) {
	enabled := &models.Config{LiveDemo: &models.LiveDemoConfig{Enabled: true}}

	t.Setenv("DEMO_MODE", "")
	if demoModeEnabled(enabled) {
		t.Error("demo API must be off without DEMO_MODE")
	}
	t.Setenv("DEMO_MODE", "true")
	if !demoModeEnabled(enabled) {
		t.Error("demo API must be on with DEMO_MODE=true and live_demo.enabled")
	}
	if demoModeEnabled(&models.Config{}) || demoModeEnabled(&models.Config{LiveDemo: &models.LiveDemoConfig{}}) || demoModeEnabled(nil) {
		t.Error("demo API must be off without an enabled live_demo section")
	}
}

func readEvent(t *testing.T, r *bufio.Reader) string {
	t.Helper()
	type result struct {
		s   string
		err error
	}
	ch := make(chan result, 1)
	go func() {
		var sb strings.Builder
		for {
			line, err := r.ReadString('\n')
			sb.WriteString(line)
			if err != nil {
				ch <- result{sb.String(), err}
				return
			}
			if line == "\n" && sb.Len() > 1 {
				ch <- result{sb.String(), nil}
				return
			}
		}
	}()
	select {
	case res := <-ch:
		if res.err != nil {
			t.Fatalf("reading event failed: %v (got %q)", res.err, res.s)
		}
		return res.s
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for an SSE event")
		return ""
	}
}
