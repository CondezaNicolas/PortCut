package domain

import (
	"reflect"
	"testing"
)

func TestNewPortProcessEntryAppliesNormalizationDefaults(t *testing.T) {
	entry := NewPortProcessEntry(PortProcessEntryInput{Port: 8080, PID: 4242})

	if entry.Protocol != ProtocolTCP {
		t.Fatalf("expected default protocol %q, got %q", ProtocolTCP, entry.Protocol)
	}

	if entry.State != StateListening {
		t.Fatalf("expected default state %q, got %q", StateListening, entry.State)
	}

	if entry.ID != "tcp:listening:8080:4242" {
		t.Fatalf("unexpected normalized id %q", entry.ID)
	}
}

func TestPortProcessEntryUsesMissingMetadataPlaceholders(t *testing.T) {
	entry := NewPortProcessEntry(PortProcessEntryInput{Port: 3000})

	if got := entry.DisplayProcessName(); got != UnknownProcessName {
		t.Fatalf("expected placeholder name %q, got %q", UnknownProcessName, got)
	}

	if got := entry.DisplayPID(); got != UnknownPIDLabel {
		t.Fatalf("expected placeholder pid label %q, got %q", UnknownPIDLabel, got)
	}

	if entry.CanTerminate() {
		t.Fatal("expected entry without pid to be non-terminable")
	}
}

func TestNewPortProcessEntryTrimsProcessName(t *testing.T) {
	entry := NewPortProcessEntry(PortProcessEntryInput{Port: 8080, PID: 4242, ProcessName: "  node  "})

	if entry.ProcessName != "node" {
		t.Fatalf("expected trimmed process name, got %q", entry.ProcessName)
	}
}

func TestDeduplicateKillTargetsCollapsesRowsByPID(t *testing.T) {
	entries := []PortProcessEntry{
		NewPortProcessEntry(PortProcessEntryInput{Port: 3000, PID: 100, ProcessName: "api"}),
		NewPortProcessEntry(PortProcessEntryInput{Port: 3001, PID: 100, ProcessName: "api"}),
		NewPortProcessEntry(PortProcessEntryInput{Port: 4000, PID: 200, ProcessName: "worker"}),
		NewPortProcessEntry(PortProcessEntryInput{Port: 5000}),
	}

	targets := DeduplicateKillTargets(entries)

	if len(targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(targets))
	}

	if targets[0].PID != 100 {
		t.Fatalf("expected first pid 100, got %d", targets[0].PID)
	}

	wantPorts := []uint16{3000, 3001}
	if !reflect.DeepEqual(targets[0].Ports, wantPorts) {
		t.Fatalf("expected merged ports %v, got %v", wantPorts, targets[0].Ports)
	}

	if targets[1].PID != 200 {
		t.Fatalf("expected second pid 200, got %d", targets[1].PID)
	}
}
