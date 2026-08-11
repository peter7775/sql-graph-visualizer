package api

import "testing"

func TestResolveBenchmarkRequest(t *testing.T) {
	tests := []struct {
		name         string
		req          BenchmarkRequest
		wantTool     string
		wantTestType string
	}{
		{
			name:     "explicit tool wins over benchmark_type",
			req:      BenchmarkRequest{Tool: "custom", BenchmarkType: "sysbench"},
			wantTool: "custom",
		},
		{
			name:     "legacy benchmark_type carrying a tool name",
			req:      BenchmarkRequest{BenchmarkType: "sysbench"},
			wantTool: "sysbench",
		},
		{
			name:     "legacy benchmark_type carrying a tool name (custom)",
			req:      BenchmarkRequest{BenchmarkType: "custom"},
			wantTool: "custom",
		},
		{
			name:         "legacy benchmark_type carrying a sysbench test type",
			req:          BenchmarkRequest{BenchmarkType: "oltp_read_write"},
			wantTool:     "sysbench",
			wantTestType: "oltp_read_write",
		},
		{
			name:         "explicit test type is preserved alongside legacy benchmark_type",
			req:          BenchmarkRequest{BenchmarkType: "oltp_read_write", TestType: "oltp_point_select"},
			wantTool:     "sysbench",
			wantTestType: "oltp_point_select",
		},
		{
			name:     "no fields set defaults to sysbench",
			req:      BenchmarkRequest{},
			wantTool: "sysbench",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tool, testType := resolveBenchmarkRequest(tt.req)
			if tool != tt.wantTool {
				t.Errorf("resolveBenchmarkRequest() tool = %q, want %q", tool, tt.wantTool)
			}
			if testType != tt.wantTestType {
				t.Errorf("resolveBenchmarkRequest() testType = %q, want %q", testType, tt.wantTestType)
			}
		})
	}
}
