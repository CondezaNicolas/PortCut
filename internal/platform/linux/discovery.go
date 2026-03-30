package linux

import (
	"regexp"
	"strconv"
	"strings"

	"portcut/internal/domain"
)

const CommandName = "ss"

var processPattern = regexp.MustCompile(`\("([^"]*)",pid=(\d+)`)

func CommandArgs() []string {
	return []string{"-H", "-ltnp"}
}

func ParseDiscoveryOutput(output []byte) ([]domain.PortProcessEntry, error) {
	var entries []domain.PortProcessEntry

	for _, rawLine := range strings.Split(string(output), "\n") {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		if !strings.EqualFold(fields[0], "LISTEN") {
			continue
		}

		port, ok := parsePort(fields[3])
		if !ok {
			continue
		}

		processes := parseProcesses(line)
		if len(processes) == 0 {
			entries = append(entries, domain.NewPortProcessEntry(domain.PortProcessEntryInput{Port: port}))
			continue
		}

		for _, process := range processes {
			entries = append(entries, domain.NewPortProcessEntry(domain.PortProcessEntryInput{
				Port:        port,
				PID:         process.pid,
				ProcessName: process.name,
			}))
		}
	}

	return entries, nil
}

type processRecord struct {
	pid  int
	name string
}

func parseProcesses(line string) []processRecord {
	matches := processPattern.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return nil
	}

	processes := make([]processRecord, 0, len(matches))
	for _, match := range matches {
		pid, err := strconv.Atoi(match[2])
		if err != nil {
			continue
		}

		processes = append(processes, processRecord{
			pid:  pid,
			name: strings.TrimSpace(match[1]),
		})
	}

	return processes
}

func parsePort(address string) (uint16, bool) {
	index := strings.LastIndex(address, ":")
	if index == -1 || index == len(address)-1 {
		return 0, false
	}

	value, err := strconv.ParseUint(address[index+1:], 10, 16)
	if err != nil {
		return 0, false
	}

	return uint16(value), true
}
