package windows

import (
	"bytes"
	"encoding/json"
	"strconv"
	"strings"

	"portcut/internal/domain"
)

const CommandName = "powershell"

const discoveryScript = `$connections = Get-NetTCPConnection -State Listen | ForEach-Object {
	$process = $null
	try {
		$process = Get-Process -Id $_.OwningProcess -ErrorAction Stop
	} catch {
	}
	[pscustomobject]@{
		State = $_.State.ToString()
		LocalPort = $_.LocalPort
		OwningProcess = $_.OwningProcess
		ProcessName = if ($process) { $process.ProcessName } else { $null }
	}
}
$connections | ConvertTo-Json -Compress`

type rawEntry struct {
	State         string          `json:"State"`
	LocalPort     json.RawMessage `json:"LocalPort"`
	OwningProcess json.RawMessage `json:"OwningProcess"`
	ProcessName   *string         `json:"ProcessName"`
}

func CommandArgs() []string {
	return []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", discoveryScript}
}

func ParseDiscoveryOutput(output []byte) ([]domain.PortProcessEntry, error) {
	trimmed := bytes.TrimSpace(output)
	if len(trimmed) == 0 || bytes.Equal(trimmed, []byte("null")) {
		return nil, nil
	}

	var rawEntries []rawEntry
	if trimmed[0] == '[' {
		if err := json.Unmarshal(trimmed, &rawEntries); err != nil {
			return nil, err
		}
	} else {
		var entry rawEntry
		if err := json.Unmarshal(trimmed, &entry); err != nil {
			return nil, err
		}
		rawEntries = []rawEntry{entry}
	}

	entries := make([]domain.PortProcessEntry, 0, len(rawEntries))
	for _, raw := range rawEntries {
		if raw.State != "" && !strings.EqualFold(raw.State, "Listen") {
			continue
		}

		port, ok := parseInt(raw.LocalPort)
		if !ok || port <= 0 || port > 65535 {
			continue
		}

		pid, _ := parseInt(raw.OwningProcess)
		name := ""
		if raw.ProcessName != nil {
			name = strings.TrimSpace(*raw.ProcessName)
		}

		entries = append(entries, domain.NewPortProcessEntry(domain.PortProcessEntryInput{
			Port:        uint16(port),
			PID:         pid,
			ProcessName: name,
		}))
	}

	return entries, nil
}

func parseInt(raw json.RawMessage) (int, bool) {
	if len(raw) == 0 {
		return 0, false
	}

	var number int
	if err := json.Unmarshal(raw, &number); err == nil {
		return number, true
	}

	var text string
	if err := json.Unmarshal(raw, &text); err != nil {
		return 0, false
	}

	value, err := strconv.Atoi(strings.TrimSpace(text))
	if err != nil {
		return 0, false
	}

	return value, true
}
