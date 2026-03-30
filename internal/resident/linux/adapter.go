package linux

import (
	"strconv"
	"strings"

	residentassets "portcut/internal/resident/assets"
)

const Title = "Portcut"
const Tooltip = "Portcut resident mode"

type SupportStatus struct {
	Supported bool
	Reason    string
}

func Icon() []byte {
	return residentassets.TrayPNG()
}

func ShutdownCommand(pid int) (string, []string) {
	return "kill", []string{"-TERM", strconv.Itoa(pid)}
}

func SupportStatusFor(env map[string]string) SupportStatus {
	if !hasDisplaySession(env) {
		return SupportStatus{Reason: "graphical desktop session not detected"}
	}

	desktops := desktopTokens(env)
	if len(desktops) == 0 {
		return SupportStatus{Reason: "desktop environment does not advertise supported tray integration"}
	}

	for _, desktop := range desktops {
		if desktop == "gnome" {
			return SupportStatus{Reason: "gnome requires tray extensions that Portcut does not manage"}
		}
	}

	for _, desktop := range desktops {
		if isSupportedDesktop(desktop) {
			return SupportStatus{Supported: true}
		}
	}

	return SupportStatus{Reason: "desktop environment does not advertise supported tray integration"}
}

func hasDisplaySession(env map[string]string) bool {
	return strings.TrimSpace(env["DISPLAY"]) != "" || strings.TrimSpace(env["WAYLAND_DISPLAY"]) != ""
}

func desktopTokens(env map[string]string) []string {
	values := []string{env["XDG_CURRENT_DESKTOP"], env["XDG_SESSION_DESKTOP"], env["DESKTOP_SESSION"]}
	seen := make(map[string]struct{}, len(values))
	tokens := make([]string, 0, len(values))
	for _, value := range values {
		for _, part := range strings.Split(value, ":") {
			token := strings.ToLower(strings.TrimSpace(part))
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			tokens = append(tokens, token)
		}
	}

	return tokens
}

func isSupportedDesktop(token string) bool {
	switch token {
	case "kde", "plasma", "xfce", "xfce4", "lxqt", "mate", "cinnamon", "unity", "pantheon", "budgie", "ukui", "deepin":
		return true
	default:
		return false
	}
}
