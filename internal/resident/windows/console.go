package windows

import (
	"fmt"
	"os/exec"
	"strings"
)

func NewConsoleCommand(executable string) *exec.Cmd {
	name, args := FreshConsoleCommand(executable)
	return exec.Command(name, args...)
}

func FreshConsoleCommand(executable string) (string, []string) {
	command := fmt.Sprintf("$process = Start-Process -FilePath '%s' -PassThru -WindowStyle Normal; [Console]::Out.WriteLine($process.Id); $process.WaitForExit(); exit $process.ExitCode", escapePowerShellSingleQuoted(executable))
	return "powershell", []string{"-NoProfile", "-NonInteractive", "-ExecutionPolicy", "Bypass", "-Command", command}
}

func escapePowerShellSingleQuoted(value string) string {
	return strings.ReplaceAll(value, "'", "''")
}
