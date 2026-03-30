package windows

import (
	"encoding/binary"
	"fmt"
	"strings"
	"testing"
)

func TestForegroundCommandRequestsWindowActivation(t *testing.T) {
	name, args := ForegroundCommand(4242)
	if name != "powershell" {
		t.Fatalf("expected powershell foreground command, got %q", name)
	}
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "AppActivate(4242)") {
		t.Fatalf("expected AppActivate pid in command, got %q", joined)
	}
	if !strings.Contains(joined, "-ExecutionPolicy Bypass") {
		t.Fatalf("expected powershell execution policy flags, got %q", joined)
	}
}

func TestShutdownCommandRequestsGracefulCloseBeforeStop(t *testing.T) {
	name, args := ShutdownCommand(4242)
	if name != "powershell" {
		t.Fatalf("expected powershell shutdown command, got %q", name)
	}
	joined := strings.Join(args, " ")
	for _, want := range []string{"Get-Process -Id 4242", "$process.CloseMainWindow()", "$process.WaitForExit(1500)", "Stop-Process -Id 4242"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in shutdown command, got %q", want, joined)
		}
	}
}

func TestIconDecodesForTrayUsage(t *testing.T) {
	icon := Icon()
	if len(icon) == 0 {
		t.Fatal("expected tray icon bytes")
	}

	if err := validateICOPayload(icon); err != nil {
		t.Fatalf("expected valid ico payload, got %v", err)
	}

	reserved := binary.LittleEndian.Uint16(icon[0:2])
	if reserved != 0 {
		t.Fatalf("expected ico reserved field to be zero, got %d", reserved)
	}

	iconType := binary.LittleEndian.Uint16(icon[2:4])
	if iconType != 1 {
		t.Fatalf("expected ico type 1, got %d", iconType)
	}

	count := binary.LittleEndian.Uint16(icon[4:6])
	if count < 1 {
		t.Fatalf("expected at least one ico image entry, got %d", count)
	}
}

func TestValidateICOPayloadRejectsMalformedData(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    string
	}{
		{
			name:    "truncated header",
			payload: []byte{0x00, 0x00, 0x01, 0x00, 0x01},
			want:    "too short for ico header",
		},
		{
			name:    "wrong reserved field",
			payload: append([]byte(nil), validICOPayloadForTest()...),
			want:    "reserved field",
		},
		{
			name:    "wrong icon type",
			payload: append([]byte(nil), validICOPayloadForTest()...),
			want:    "type field",
		},
		{
			name:    "missing directory entries",
			payload: []byte{0x00, 0x00, 0x01, 0x00, 0x01, 0x00},
			want:    "directory entries",
		},
		{
			name:    "image offset out of bounds",
			payload: append([]byte(nil), validICOPayloadForTest()...),
			want:    "outside payload bounds",
		},
		{
			name:    "image size out of bounds",
			payload: append([]byte(nil), validICOPayloadForTest()...),
			want:    "outside payload bounds",
		},
	}

	tests[1].payload[0] = 0x01
	tests[2].payload[2] = 0x02
	binary.LittleEndian.PutUint32(tests[4].payload[18:22], uint32(len(tests[4].payload)+1))
	binary.LittleEndian.PutUint32(tests[5].payload[14:18], uint32(len(tests[5].payload)+1))

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateICOPayload(tt.payload)
			if err == nil {
				t.Fatal("expected ico validation to fail")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("expected error containing %q, got %v", tt.want, err)
			}
		})
	}
}

func validateICOPayload(payload []byte) error {
	if len(payload) < 6 {
		return fmt.Errorf("payload too short for ico header: %d", len(payload))
	}

	reserved := binary.LittleEndian.Uint16(payload[0:2])
	if reserved != 0 {
		return fmt.Errorf("invalid ico reserved field: %d", reserved)
	}

	iconType := binary.LittleEndian.Uint16(payload[2:4])
	if iconType != 1 {
		return fmt.Errorf("invalid ico type field: %d", iconType)
	}

	count := int(binary.LittleEndian.Uint16(payload[4:6]))
	if count < 1 {
		return fmt.Errorf("invalid ico image count: %d", count)
	}

	directoryBytes := count * 16
	if len(payload) < 6+directoryBytes {
		return fmt.Errorf("payload truncated for directory entries: need %d bytes, got %d", 6+directoryBytes, len(payload))
	}

	for i := 0; i < count; i++ {
		entryOffset := 6 + (i * 16)
		entry := payload[entryOffset : entryOffset+16]
		imageSize := binary.LittleEndian.Uint32(entry[8:12])
		imageOffset := binary.LittleEndian.Uint32(entry[12:16])
		if imageSize == 0 {
			return fmt.Errorf("ico entry %d has zero image size", i)
		}
		if imageOffset >= uint32(len(payload)) {
			return fmt.Errorf("ico entry %d starts outside payload bounds: offset=%d len=%d", i, imageOffset, len(payload))
		}
		if imageSize > uint32(len(payload))-imageOffset {
			return fmt.Errorf("ico entry %d extends outside payload bounds: offset=%d size=%d len=%d", i, imageOffset, imageSize, len(payload))
		}
	}

	return nil
}

func validICOPayloadForTest() []byte {
	return []byte{
		0x00, 0x00, 0x01, 0x00, 0x01, 0x00,
		0x10, 0x10, 0x00, 0x00, 0x01, 0x00, 0x20, 0x00,
		0x04, 0x00, 0x00, 0x00,
		0x16, 0x00, 0x00, 0x00,
		0x89, 0x50, 0x4e, 0x47,
	}
}
