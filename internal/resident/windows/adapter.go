package windows

import (
	"fmt"

	residentassets "portcut/internal/resident/assets"
)

const Title = "Portcut"
const Tooltip = "Portcut resident mode"

func Icon() []byte {
	return residentassets.TrayICO()
}

func ForegroundCommand(pid int) (string, []string) {
	command := fmt.Sprintf("Add-Type -AssemblyName Microsoft.VisualBasic; [Microsoft.VisualBasic.Interaction]::AppActivate(%d) | Out-Null", pid)
	return "powershell", []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command}
}

func ShutdownCommand(pid int) (string, []string) {
	command := fmt.Sprintf("$process = Get-Process -Id %d -ErrorAction SilentlyContinue; if ($null -eq $process) { exit 0 }; if ($process.CloseMainWindow()) { if ($process.WaitForExit(1500)) { exit 0 } }; Stop-Process -Id %d -ErrorAction Stop", pid, pid)
	return "powershell", []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command}
}
