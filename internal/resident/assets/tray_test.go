package assets

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"testing"
)

func TestTrayPNGReturnsStableClonedPayload(t *testing.T) {
	first := TrayPNG()
	second := TrayPNG()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("expected tray png bytes")
	}
	if !bytes.Equal(first, second) {
		t.Fatal("expected repeated png accessor calls to return the same payload")
	}
	if &first[0] == &second[0] {
		t.Fatal("expected png accessor to clone per call")
	}
	if !bytes.Equal(first[:8], []byte{0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a}) {
		t.Fatal("expected png signature")
	}

	original := second[0]
	first[0] ^= 0xff
	third := TrayPNG()
	if third[0] != original {
		t.Fatal("expected png mutations to stay isolated from future callers")
	}
}

func TestTrayICOReturnsStableClonedPayload(t *testing.T) {
	first := TrayICO()
	second := TrayICO()
	if len(first) == 0 || len(second) == 0 {
		t.Fatal("expected tray ico bytes")
	}
	if !bytes.Equal(first, second) {
		t.Fatal("expected repeated ico accessor calls to return the same payload")
	}
	if &first[0] == &second[0] {
		t.Fatal("expected ico accessor to clone per call")
	}
	if err := validateICOPayload(first); err != nil {
		t.Fatalf("expected valid ico payload, got %v", err)
	}

	original := second[0]
	first[0] ^= 0xff
	third := TrayICO()
	if third[0] != original {
		t.Fatal("expected ico mutations to stay isolated from future callers")
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
