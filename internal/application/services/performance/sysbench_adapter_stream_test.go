package performance

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"sql-graph-visualizer/internal/application/ports"
)

const interimFixture = "[ 10s ] thds: 8 tps: 123.45 qps: 2469.00 (r/w/o: 1728.30/493.80/246.90) lat (ms,95%): 21.50 err/s: 0.20 reconn/s: 0.00"

func TestParseSysbenchInterim(t *testing.T) {
	sample, ok := parseSysbenchInterim(interimFixture, 5)
	if !ok || sample.ElapsedSeconds != 10 || sample.Threads != 8 || sample.Interval.QPS != 2469 ||
		sample.Interval.Queries != 12345 || sample.Interval.P95LatencyMs != 21.5 || sample.Interval.Errors != 1 {
		t.Fatalf("unexpected sample %+v ok=%v", sample, ok)
	}
	if sample.Queries == nil || sample.Graph.Nodes == nil || sample.Graph.Edges == nil {
		t.Error("empty slices must serialize as []")
	}
	for _, line := range []string{"not a sample", strings.Replace(interimFixture, "2469.00", "..", 1)} {
		if _, ok := parseSysbenchInterim(line, 0); ok {
			t.Errorf("parsed malformed line %q", line)
		}
	}
	if _, ok := parseSysbenchInterim(interimFixture, 10); ok {
		t.Error("non-increasing elapsed time accepted")
	}
}

func TestSysbenchPostgreSQLArguments(t *testing.T) {
	adapter := &SysbenchAdapter{config: defaultSysbenchConfig()}
	args, err := adapter.parseDatabaseURL("postgresql://user:pass@[::1]:55432/bench?sslmode=disable", "postgresql")
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"--db-driver=pgsql", "--pgsql-user=user", "--pgsql-password=pass", "--pgsql-host=::1", "--pgsql-port=55432", "--pgsql-db=bench", "--pgsql-sslmode=disable"} {
		if !strings.Contains(joined, want) {
			t.Errorf("missing %q", want)
		}
	}
	config := ports.BenchmarkConfig{DatabaseURL: "postgresql://localhost/bench", DatabaseType: "postgresql", TestType: "oltp_read_only"}
	cmd, err := adapter.buildCommand(config)
	if err != nil {
		t.Fatal(err)
	}
	if joined := strings.Join(cmd.Args, " "); !strings.Contains(joined, "--report-interval=1") || strings.Contains(joined, "--pgsql-schema") {
		t.Fatalf("invalid default options: %s", joined)
	}
	adapter.config.PostgreSQLDefaults.Schema = "custom"
	if _, err := adapter.buildCommand(config); err == nil {
		t.Error("non-default schema must not produce an unsupported sysbench CLI flag")
	}
}

func fakeSysbench(t *testing.T, body string) *SysbenchAdapter {
	t.Helper()
	path := filepath.Join(t.TempDir(), "sysbench")
	script := "#!/bin/sh\nif [ \"$1\" = --version ]; then printf 'sysbench 1.0.20\\n'; exit 0; fi\n" + body
	if err := os.WriteFile(path, []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return NewSysbenchAdapter(newTestLogger(), &SysbenchConfig{BinaryPath: path, ReportInterval: 5})
}

func TestSysbenchExecuteStream(t *testing.T) {
	adapter := fakeSysbench(t, "printf '%s\\n' '"+interimFixture+"'\nprintf 'queries: 12345 (2469.00 queries/sec)\\navg: 5.0\\n95th percentile: 21.50\\n'\n")
	config := ports.BenchmarkConfig{DatabaseURL: "postgresql://localhost/db", DatabaseType: "postgresql", TestType: "oltp_read_only", Threads: 8, Duration: 10 * time.Second}
	cmd, err := adapter.buildCommand(config)
	if err != nil || !strings.Contains(strings.Join(cmd.Args, " "), "--report-interval=5") {
		t.Fatalf("command=%v err=%v", cmd, err)
	}
	samples := []ports.LiveSample{}
	result, err := adapter.ExecuteStream(context.Background(), config, func(sample ports.LiveSample) { samples = append(samples, sample) })
	if err != nil || len(samples) != 1 || result.Metrics.QueriesPerSecond != 2469 || len(result.QueryResults) != 0 {
		t.Fatalf("result=%+v samples=%+v err=%v", result, samples, err)
	}
	if _, err := adapter.Execute(context.Background(), config); err != nil {
		t.Fatal(err)
	}
}

func TestSysbenchStreamCancellation(t *testing.T) {
	adapter := fakeSysbench(t, "exec sleep 30\n")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	_, err := adapter.ExecuteStream(ctx, ports.BenchmarkConfig{DatabaseURL: "mysql://localhost/db"}, nil)
	if !errors.Is(err, context.DeadlineExceeded) || time.Since(start) > 2*time.Second {
		t.Fatalf("cancellation took %v err=%v", time.Since(start), err)
	}
}
