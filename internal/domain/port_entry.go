package domain

import (
	"fmt"
	"slices"
	"strings"
)

const (
	UnknownProcessName = "unknown process"
	UnknownPIDLabel    = "n/a"
)

type NetworkProtocol string

const (
	ProtocolUnknown NetworkProtocol = "unknown"
	ProtocolTCP     NetworkProtocol = "tcp"
	ProtocolUDP     NetworkProtocol = "udp"
)

type SocketState string

const (
	StateUnknown   SocketState = "unknown"
	StateListening SocketState = "listening"
	StateClosed    SocketState = "closed"
)

type PortProcessEntryInput struct {
	Port        uint16
	Protocol    NetworkProtocol
	State       SocketState
	PID         int
	ProcessName string
}

type PortProcessEntry struct {
	ID          string
	Port        uint16
	Protocol    NetworkProtocol
	State       SocketState
	PID         int
	ProcessName string
}

type KillTarget struct {
	PID         int
	ProcessName string
	Ports       []uint16
}

func NewPortProcessEntry(input PortProcessEntryInput) PortProcessEntry {
	protocol := input.Protocol
	if protocol == "" {
		protocol = ProtocolTCP
	}

	state := input.State
	if state == "" {
		state = StateListening
	}

	name := strings.TrimSpace(input.ProcessName)

	entry := PortProcessEntry{
		Port:        input.Port,
		Protocol:    protocol,
		State:       state,
		PID:         input.PID,
		ProcessName: name,
	}
	entry.ID = entryKey(entry)

	return entry
}

func (e PortProcessEntry) DisplayProcessName() string {
	if e.ProcessName == "" {
		return UnknownProcessName
	}

	return e.ProcessName
}

func (e PortProcessEntry) DisplayPID() string {
	if e.PID <= 0 {
		return UnknownPIDLabel
	}

	return fmt.Sprintf("%d", e.PID)
}

func (e PortProcessEntry) CanTerminate() bool {
	return e.PID > 0
}

func (e PortProcessEntry) KillTarget() (KillTarget, bool) {
	if !e.CanTerminate() {
		return KillTarget{}, false
	}

	return KillTarget{
		PID:         e.PID,
		ProcessName: e.DisplayProcessName(),
		Ports:       []uint16{e.Port},
	}, true
}

func entryKey(entry PortProcessEntry) string {
	return fmt.Sprintf("%s:%s:%d:%d", entry.Protocol, entry.State, entry.Port, entry.PID)
}

func mergeKillTargetPorts(ports []uint16, next uint16) []uint16 {
	if !slices.Contains(ports, next) {
		ports = append(ports, next)
		slices.Sort(ports)
	}

	return ports
}
