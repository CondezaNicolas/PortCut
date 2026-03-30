package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

const (
	sourcePath = "assets/resident/portcut-tray.svg"
	pngPath    = "internal/resident/assets/portcut-tray.png"
	icoPath    = "internal/resident/assets/portcut-tray.ico"
	iconSize   = 256
)

func main() {
	root, err := os.Getwd()
	if err != nil {
		fail("resolve working directory", err)
	}

	svgBytes, err := os.ReadFile(filepath.Join(root, sourcePath))
	if err != nil {
		fail("read tray svg", err)
	}

	pngBytes, err := renderPNG(svgBytes, iconSize)
	if err != nil {
		fail("render tray png", err)
	}

	if err := writeFile(filepath.Join(root, pngPath), pngBytes); err != nil {
		fail("write tray png", err)
	}

	if err := writeFile(filepath.Join(root, icoPath), encodeICO(pngBytes)); err != nil {
		fail("write tray ico", err)
	}
}

func renderPNG(svgBytes []byte, size int) ([]byte, error) {
	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgBytes))
	if err != nil {
		return nil, err
	}

	icon.SetTarget(0, 0, float64(size), float64(size))
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	scanner := rasterx.NewScannerGV(size, size, img, img.Bounds())
	dasher := rasterx.NewDasher(size, size, scanner)
	icon.Draw(dasher, 1)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func encodeICO(pngBytes []byte) []byte {
	header := make([]byte, 6)
	binary.LittleEndian.PutUint16(header[2:4], 1)
	binary.LittleEndian.PutUint16(header[4:6], 1)

	entry := make([]byte, 16)
	binary.LittleEndian.PutUint16(entry[4:6], 1)
	binary.LittleEndian.PutUint16(entry[6:8], 32)
	binary.LittleEndian.PutUint32(entry[8:12], uint32(len(pngBytes)))
	binary.LittleEndian.PutUint32(entry[12:16], uint32(len(header)+len(entry)))

	payload := make([]byte, 0, len(header)+len(entry)+len(pngBytes))
	payload = append(payload, header...)
	payload = append(payload, entry...)
	payload = append(payload, pngBytes...)

	return payload
}

func writeFile(path string, contents []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, contents, 0o644)
}

func fail(action string, err error) {
	fmt.Fprintf(os.Stderr, "%s: %v\n", action, err)
	os.Exit(1)
}
