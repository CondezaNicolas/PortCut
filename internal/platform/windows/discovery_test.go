package windows

import "testing"

func TestParseDiscoveryOutputNormalizesJSONRows(t *testing.T) {
	output := []byte(`[
		{"State":"Listen","LocalPort":3000,"OwningProcess":1234,"ProcessName":"node"},
		{"State":"Established","LocalPort":4000,"OwningProcess":555,"ProcessName":"curl"},
		{"State":"Listen","LocalPort":"8080","OwningProcess":"4321","ProcessName":null},
		{"State":"Listen","LocalPort":9000,"OwningProcess":0,"ProcessName":"  "}
	]`)

	entries, err := ParseDiscoveryOutput(output)
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].Port != 3000 || entries[0].PID != 1234 || entries[0].ProcessName != "node" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}

	if entries[1].Port != 8080 || entries[1].PID != 4321 || entries[1].ProcessName != "" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}

	if entries[2].Port != 9000 || entries[2].PID != 0 || entries[2].ProcessName != "" {
		t.Fatalf("unexpected third entry: %+v", entries[2])
	}
}

func TestParseDiscoveryOutputAcceptsSingleObject(t *testing.T) {
	entries, err := ParseDiscoveryOutput([]byte(`{"State":"Listen","LocalPort":5000,"OwningProcess":77,"ProcessName":"api"}`))
	if err != nil {
		t.Fatalf("expected parse success, got %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}

	if entries[0].Port != 5000 || entries[0].PID != 77 || entries[0].ProcessName != "api" {
		t.Fatalf("unexpected entry: %+v", entries[0])
	}
}
