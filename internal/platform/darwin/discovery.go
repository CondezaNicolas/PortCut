package darwin

import (
	"strconv"
	"strings"

	"portcut/internal/domain"
)

const CommandName = "lsof"

func CommandArgs() []string {
	return []string{"-nP", "-iTCP", "-sTCP:LISTEN", "-Fpcn"}
}

func ParseDiscoveryOutput(output []byte) ([]domain.PortProcessEntry, error) {
	var entries []domain.PortProcessEntry
	currentPID := 0
	currentName := ""

	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		switch line[0] {
		case 'p':
			currentPID = parsePID(line[1:])
			currentName = ""
		case 'c':
			currentName = strings.TrimSpace(line[1:])
		case 'n':
			port, ok := parsePort(line[1:])
			if !ok {
				continue
			}

			entries = append(entries, domain.NewPortProcessEntry(domain.PortProcessEntryInput{
				Port:        port,
				PID:         currentPID,
				ProcessName: currentName,
			}))
		}
	}

	return entries, nil
}

func parsePID(value string) int {
	pid, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return 0
	}

	return pid
}

func parsePort(endpoint string) (uint16, bool) {
	trimmed := strings.TrimSpace(strings.TrimSuffix(endpoint, " (LISTEN)"))
	index := strings.LastIndex(trimmed, ":")
	if index == -1 || index == len(trimmed)-1 {
		return 0, false
	}

	value, err := strconv.ParseUint(trimmed[index+1:], 10, 16)
	if err != nil {
		return 0, false
	}

	return uint16(value), true
}
