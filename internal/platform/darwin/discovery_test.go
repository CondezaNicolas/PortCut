package darwin

import "testing"

func TestParseDiscoveryOutputPreservesRowsWithMissingProcessNames(t *testing.T) {
	output := []byte("p123\ncControlCenter\nn*:7000\np456\nn127.0.0.1:8080\npbad\ncignored\nn[::1]:9000\n")

	entries, err := ParseDiscoveryOutput(output)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].PID != 123 || entries[0].ProcessName != "ControlCenter" || entries[0].Port != 7000 {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}

	if entries[1].PID != 456 || entries[1].ProcessName != "" || entries[1].Port != 8080 {
		t.Fatalf("expected missing process name to be preserved, got %+v", entries[1])
	}

	if entries[2].PID != 0 || entries[2].ProcessName != "ignored" || entries[2].Port != 9000 {
		t.Fatalf("expected invalid pid fallback, got %+v", entries[2])
	}
}
