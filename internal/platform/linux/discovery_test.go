package linux

import "testing"

func TestParseDiscoveryOutputKeepsListeningTCPRows(t *testing.T) {
	output := []byte("LISTEN 0 128 127.0.0.1:3000 0.0.0.0:* users:((\"api\",pid=1234,fd=7))\nESTAB 0 0 127.0.0.1:3001 127.0.0.1:4000 users:((\"curl\",pid=999,fd=9))\nLISTEN 0 128 [::]:8080 [::]:* users:((\"worker\",pid=4321,fd=8),(\"helper\",pid=4322,fd=9))\nLISTEN 0 128 *:9000 *:*\n")

	entries, err := ParseDiscoveryOutput(output)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	if entries[0].Port != 3000 || entries[0].PID != 1234 || entries[0].ProcessName != "api" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}

	if entries[1].Port != 8080 || entries[1].PID != 4321 || entries[1].ProcessName != "worker" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}

	if entries[2].Port != 8080 || entries[2].PID != 4322 || entries[2].ProcessName != "helper" {
		t.Fatalf("unexpected third entry: %+v", entries[2])
	}

	if entries[3].Port != 9000 || entries[3].PID != 0 || entries[3].ProcessName != "" {
		t.Fatalf("unexpected metadata fallback entry: %+v", entries[3])
	}
}
