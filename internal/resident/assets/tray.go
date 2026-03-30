package assets

import (
	_ "embed"
	"slices"
)

//go:embed portcut-tray.png
var trayPNG []byte

//go:embed portcut-tray.ico
var trayICO []byte

func TrayPNG() []byte {
	return slices.Clone(trayPNG)
}

func TrayICO() []byte {
	return slices.Clone(trayICO)
}
