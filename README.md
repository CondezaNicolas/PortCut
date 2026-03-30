# Portcut

A cross-platform terminal UI for managing listening TCP ports. List, inspect, and terminate processes occupying ports with a safe, review-first workflow.

[![Go Version](https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey)](https://github.com)

## Features

- **Cross-platform**: Works on Windows, macOS, and Linux
- **Category-first browsing**: Ports grouped by process type (Node, System, Docker, Databases, etc.)
- **Multi-selection**: Select multiple ports and terminate them together
- **Safe workflow**: Review and confirm before any destructive action
- **Deduplication**: Automatically deduplicates selections by PID
- **Resident mode**: System tray/menu bar integration for quick access
- **Windows detached mode**: Tray survives terminal closure on Windows
- **Keyboard-driven**: Full keyboard navigation with intuitive shortcuts

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/yourusername/portcut.git
cd portcut

# Build the TUI
go build -o portcut ./cmd/portcut

# Build the resident mode (optional)
go build -o portcut-resident ./cmd/portcut-resident

# Generate branded tray assets (optional, for development)
go run ./scripts/generate_resident_assets.go
```

### Prerequisites

- **Go 1.21+** for building from source
- **PowerShell** on Windows for port discovery and process termination
- A terminal that supports TUI applications

## Quick Start

### Basic Usage

```bash
# Run the TUI directly
./portcut
```

### Resident Mode (System Tray)

```bash
# Run in resident mode with system tray integration
./portcut-resident
```

On Windows, once the tray icon appears, you can close the terminal and the resident mode will continue running. Use the tray menu to open Portcut whenever you need it.

## Keyboard Shortcuts

### Category List View

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate categories |
| `Enter` | Open selected category |
| `q` | Quit |

### Category Detail View

| Key | Action |
|-----|--------|
| `↑` / `↓` | Navigate ports |
| `Space` | Toggle selection on focused row |
| `Enter` | Review graceful termination (Linux/macOS) |
| `f` | Review force termination |
| `r` | Refresh port inventory |
| `Esc` | Return to category list |
| `q` | Quit |

### Review/Confirmation Screen

| Key | Action |
|-----|--------|
| `y` | Confirm and execute termination |
| `n` / `Esc` | Cancel and return to detail view |

### Resident Mode (Tray)

| Action | Description |
|--------|-------------|
| `Open Portcut` | Launch or focus the TUI |
| `Quit` | Exit resident mode |

## Categories

Ports are automatically grouped into categories based on process names:

| Category | Examples |
|----------|----------|
| **Node / JS** | `node`, `npm`, `yarn`, `vite`, `next` |
| **System** | `systemd`, `launchd`, `svchost` |
| **Containers / WSL** | `docker`, `wslrelay`, `containerd` |
| **Databases** | `postgres`, `mysql`, `mongod`, `redis-server` |
| **Browsers** | `chrome`, `firefox`, `edge` |
| **Servers / Proxies** | `nginx`, `apache`, `caddy`, `haproxy` |
| **Other** | Processes not matching other categories |
| **Unknown** | Processes with missing/unusable names |

## Platform Support

### Windows

- Full TUI support with PowerShell
- **Force termination only** (graceful termination not supported)
- **Detached resident mode**: Tray survives terminal closure
- Fresh console window for each `Open Portcut` action

### macOS

- Full TUI support via `lsof`
- Graceful and force termination supported
- Menu bar integration for resident mode
- Detached persistence not yet supported

### Linux

- Full TUI support via `ss` / `lsof`
- Graceful and force termination supported
- Tray/status-notifier integration on supported desktops:
  - KDE Plasma
  - Xfce
  - LXQt
  - MATE
  - Cinnamon
  - Budgie
  - Unity
  - Pantheon
  - UKUI
  - Deepin

**Not supported on Linux:**
- GNOME (requires shell extensions not managed by Portcut)
- Headless sessions (no `DISPLAY` or `WAYLAND_DISPLAY`)

## Architecture

```
portcut/
├── cmd/
│   ├── portcut/              # TUI entrypoint
│   │   └── main.go
│   └── portcut-resident/     # Resident mode entrypoint
│       └── main.go
├── internal/
│   ├── app/                  # Application workflow
│   │   ├── workflow.go       # Refresh/review/execute logic
│   │   └── launcher.go       # TUI bootstrap
│   ├── domain/               # Domain models
│   │   ├── port_entry.go     # Port entry model
│   │   ├── selection.go      # Selection/dedup logic
│   │   └── category.go       # Category classification
│   ├── platform/             # Platform adapters
│   │   ├── service.go        # Platform service interface
│   │   ├── factory.go        # Platform detection
│   │   ├── linux/            # Linux discovery/termination
│   │   ├── darwin/           # macOS discovery/termination
│   │   └── windows/          # Windows discovery/termination
│   ├── resident/             # Resident mode
│   │   ├── runtime.go        # Runtime composition
│   │   ├── host.go           # Tray host controller
│   │   ├── session.go        # Session tracking
│   │   ├── session_process.go # Child process management
│   │   ├── assets/           # Branded tray assets
│   │   ├── windows/          # Windows-specific bootstrap
│   │   ├── darwin/           # macOS adapter
│   │   └── linux/            # Linux adapter
│   └── tui/                  # Bubble Tea TUI
│       └── model.go          # TUI state machine
├── assets/
│   └── resident/             # Source assets
│       └── portcut-tray.svg  # Tray icon source
└── scripts/
    └── generate_resident_assets.go  # Asset generator
```

## Safety Model

Portcut is designed with safety as a priority:

1. **Review-first**: No action is taken without explicit review and confirmation
2. **Deduplication**: Selecting multiple ports from the same process only terminates once
3. **PID validation**: Rows without a valid PID are visible but not selectable
4. **Stale data protection**: Selection is revalidated before execution
5. **Error visibility**: Failures are reported clearly, never silently ignored

## Development

### Running Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific package
go test ./internal/resident/...
```

### Generating Assets

```bash
# Generate PNG and ICO from source SVG
go run ./scripts/generate_resident_assets.go
```

### Building for Distribution

```bash
# Build for current platform
go build -o portcut ./cmd/portcut
go build -o portcut-resident ./cmd/portcut-resident

# Build for Windows (from any platform)
GOOS=windows GOARCH=amd64 go build -o portcut.exe ./cmd/portcut
GOOS=windows GOARCH=amd64 go build -o portcut-resident.exe ./cmd/portcut-resident

# Build for macOS
GOOS=darwin GOARCH=amd64 go build -o portcut ./cmd/portcut
GOOS=darwin GOARCH=amd64 go build -o portcut-resident ./cmd/portcut-resident

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o portcut ./cmd/portcut
GOOS=linux GOARCH=amd64 go build -o portcut-resident ./cmd/portcut-resident
```

## Distribution

When distributing binaries:

1. Keep `portcut` and `portcut-resident` in the **same directory**
2. The resident launcher finds the TUI executable relative to itself
3. On Windows, build with `-ldflags "-H=windowsgui"` for the resident binary to avoid console flash

## Troubleshooting

### Windows: "Unable to set icon" error

Ensure the tray icon assets are properly generated:
```bash
go run ./scripts/generate_resident_assets.go
```

### Linux: Resident mode fails to start

Check your desktop environment is supported. GNOME and headless sessions are not supported for resident mode.

### Permission denied when terminating processes

Some system processes require elevated privileges. Run with appropriate permissions or select user processes only.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add some amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## Roadmap

- [ ] Free-text search/filter within categories
- [ ] Collapse multiple ports from the same process
- [ ] Port filtering by protocol (TCP/UDP)
- [ ] Export port list to file
- [ ] Configuration file support
- [ ] Auto-refresh interval
- [ ] macOS/Linux detached persistence

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework
- [Bubbles](https://github.com/charmbracelet/bubbles) - TUI components
- [Lip Gloss](https://github.com/charmbracelet/lipgloss) - Styling
- [systray](https://github.com/getlantern/systray) - Cross-platform system tray

---

Made with ❤️ by developers who got tired of `netstat | grep` and `kill -9`
