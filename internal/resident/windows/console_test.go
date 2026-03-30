package windows

import (
	"strings"
	"testing"
)

func TestFreshConsoleCommandLaunchesPortcutInNewWindowAndReportsPID(t *testing.T) {
	name, args := FreshConsoleCommand("C:/Program Files/Portcut/portcut.exe")
	if name != "powershell" {
		t.Fatalf("expected powershell launcher, got %q", name)
	}

	joined := strings.Join(args, " ")
	for _, want := range []string{
		"Start-Process -FilePath 'C:/Program Files/Portcut/portcut.exe'",
		"-PassThru",
		"-WindowStyle Normal",
		"[Console]::Out.WriteLine($process.Id)",
		"$process.WaitForExit()",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected %q in launcher command, got %q", want, joined)
		}
	}
}

func TestFreshConsoleCommandEscapesSingleQuotesInExecutablePath(t *testing.T) {
	_, args := FreshConsoleCommand("C:/O'Brien/portcut.exe")
	joined := strings.Join(args, " ")
	if !strings.Contains(joined, "C:/O''Brien/portcut.exe") {
		t.Fatalf("expected single quote escaping in launcher command, got %q", joined)
	}
}
