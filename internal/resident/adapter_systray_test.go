package resident

import (
	"context"
	"testing"
	"time"
)

func TestSystrayAdapterOpensAndQuitsThroughMenu(t *testing.T) {
	tray := newTrayAPIDouble()
	session := &hostSessionDouble{}
	host, err := NewHost(session)
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	adapter := newSystrayAdapter(systrayAdapterConfig{
		title:   "Portcut",
		tooltip: "Portcut resident mode",
		icon:    []byte{0x1},
		api:     tray,
	})

	runDone := make(chan error, 1)
	go func() {
		runDone <- adapter.Run(context.Background(), host)
	}()

	tray.waitUntilReady(t)
	tray.click(MenuActionOpen)
	tray.waitForOpenCount(t, session, 1)
	if session.openRequests[0].Reason != OpenReasonLaunch {
		t.Fatalf("expected launch open request, got %#v", session.openRequests[0])
	}

	tray.click(MenuActionQuit)
	tray.waitForQuitCount(t, session, 1)

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("expected adapter run success, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected adapter run to finish after quit")
	}

	if tray.title != "Portcut" {
		t.Fatalf("expected tray title to be set, got %q", tray.title)
	}
	if tray.tooltip != "Portcut resident mode" {
		t.Fatalf("expected tray tooltip to be set, got %q", tray.tooltip)
	}
	if len(tray.icon) != 1 {
		t.Fatalf("expected tray icon to be set, got %d bytes", len(tray.icon))
	}
	if !tray.quitCalled {
		t.Fatal("expected tray quit to be requested")
	}
}

func TestSystrayAdapterQuitsWhenContextCancels(t *testing.T) {
	tray := newTrayAPIDouble()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	host, err := NewHost(&hostSessionDouble{})
	if err != nil {
		t.Fatalf("expected host creation success, got %v", err)
	}

	adapter := newSystrayAdapter(systrayAdapterConfig{api: tray})
	runDone := make(chan error, 1)
	go func() {
		runDone <- adapter.Run(ctx, host)
	}()

	tray.waitUntilReady(t)
	cancel()

	select {
	case err := <-runDone:
		if err != nil {
			t.Fatalf("expected adapter run success, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("expected adapter run to finish after context cancellation")
	}

	if !tray.quitCalled {
		t.Fatal("expected tray quit on context cancellation")
	}
}

type trayAPIDouble struct {
	title      string
	tooltip    string
	icon       []byte
	items      map[MenuAction]*trayMenuItemDouble
	ready      chan struct{}
	quit       chan struct{}
	quitCalled bool
}

func newTrayAPIDouble() *trayAPIDouble {
	return &trayAPIDouble{
		items: make(map[MenuAction]*trayMenuItemDouble, len(DefaultMenu())),
		ready: make(chan struct{}),
		quit:  make(chan struct{}),
	}
}

func (t *trayAPIDouble) Run(onReady func(), _ func()) {
	if onReady != nil {
		onReady()
	}
	close(t.ready)
	<-t.quit
}

func (t *trayAPIDouble) Quit() {
	if !t.quitCalled {
		t.quitCalled = true
		close(t.quit)
	}
}

func (t *trayAPIDouble) SetTitle(title string) {
	t.title = title
}

func (t *trayAPIDouble) SetTooltip(tooltip string) {
	t.tooltip = tooltip
}

func (t *trayAPIDouble) SetIcon(icon []byte) {
	t.icon = append([]byte(nil), icon...)
}

func (t *trayAPIDouble) AddMenuItem(title string, _ string) trayMenuItem {
	item := &trayMenuItemDouble{clicked: make(chan struct{}, 1)}
	for _, menuItem := range DefaultMenu() {
		if menuItem.Label == title {
			t.items[menuItem.Action] = item
			break
		}
	}

	return item
}

func (t *trayAPIDouble) click(action MenuAction) {
	item := t.items[action]
	item.clicked <- struct{}{}
}

func (t *trayAPIDouble) waitUntilReady(testingT *testing.T) {
	testingT.Helper()

	select {
	case <-t.ready:
	case <-time.After(time.Second):
		testingT.Fatal("expected tray to become ready")
	}
}

func (t *trayAPIDouble) waitForOpenCount(testingT *testing.T, session *hostSessionDouble, want int) {
	testingT.Helper()
	waitForCondition(testingT, func() bool { return len(session.openRequests) == want }, "expected tray open handler to run")
}

func (t *trayAPIDouble) waitForQuitCount(testingT *testing.T, session *hostSessionDouble, want int) {
	testingT.Helper()
	waitForCondition(testingT, func() bool { return len(session.shutdownRequests) == want }, "expected tray quit handler to run")
}

type trayMenuItemDouble struct {
	clicked chan struct{}
}

func (i *trayMenuItemDouble) Clicked() <-chan struct{} {
	return i.clicked
}

func waitForCondition(t *testing.T, check func() bool, message string) {
	t.Helper()

	deadline := time.After(time.Second)
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		if check() {
			return
		}

		select {
		case <-deadline:
			t.Fatal(message)
		case <-ticker.C:
		}
	}
}
