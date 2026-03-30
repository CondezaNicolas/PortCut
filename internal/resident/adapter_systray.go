package resident

import (
	"context"
	"errors"
	"sync"

	"github.com/getlantern/systray"
)

var ErrInvalidAdapterHost = errors.New("invalid resident adapter host")

type systrayAdapterConfig struct {
	title   string
	tooltip string
	icon    []byte
	api     trayAPI
}

type trayAPI interface {
	Run(func(), func())
	Quit()
	SetTitle(string)
	SetTooltip(string)
	SetIcon([]byte)
	AddMenuItem(string, string) trayMenuItem
}

type trayMenuItem interface {
	Clicked() <-chan struct{}
}

type systrayAdapter struct {
	config systrayAdapterConfig
}

func newSystrayAdapter(config systrayAdapterConfig) Adapter {
	if config.api == nil {
		config.api = realTrayAPI{}
	}

	return &systrayAdapter{config: config}
}

func (a *systrayAdapter) Run(ctx context.Context, host Host) error {
	if host == nil {
		return ErrInvalidAdapterHost
	}

	menu := host.Menu()
	if err := ValidateMenu(menu); err != nil {
		return err
	}

	if ctx == nil {
		ctx = context.Background()
	}

	var (
		mu     sync.Mutex
		runErr error
	)
	setErr := func(err error) {
		if err == nil {
			return
		}

		mu.Lock()
		defer mu.Unlock()
		if runErr == nil {
			runErr = err
		}
	}

	tray := a.config.api
	tray.Run(func() {
		if len(a.config.icon) > 0 {
			tray.SetIcon(a.config.icon)
		}
		if a.config.title != "" {
			tray.SetTitle(a.config.title)
		}
		if a.config.tooltip != "" {
			tray.SetTooltip(a.config.tooltip)
		}

		items := make(map[MenuAction]trayMenuItem, len(menu))
		for _, item := range menu {
			items[item.Action] = tray.AddMenuItem(item.Label, item.Label)
		}

		openItem := items[MenuActionOpen]
		quitItem := items[MenuActionQuit]

		go func() {
			for {
				select {
				case <-ctx.Done():
					tray.Quit()
					return
				case <-openItem.Clicked():
					setErr(host.OpenPortcut(context.Background()))
				case <-quitItem.Clicked():
					setErr(host.Quit(context.Background()))
					tray.Quit()
					return
				}
			}
		}()
	}, nil)

	mu.Lock()
	defer mu.Unlock()

	return runErr
}

type realTrayAPI struct{}

func (realTrayAPI) Run(onReady func(), onExit func()) {
	systray.Run(onReady, onExit)
}

func (realTrayAPI) Quit() {
	systray.Quit()
}

func (realTrayAPI) SetTitle(title string) {
	systray.SetTitle(title)
}

func (realTrayAPI) SetTooltip(tooltip string) {
	systray.SetTooltip(tooltip)
}

func (realTrayAPI) SetIcon(icon []byte) {
	systray.SetIcon(icon)
}

func (realTrayAPI) AddMenuItem(title string, tooltip string) trayMenuItem {
	return realTrayMenuItem{item: systray.AddMenuItem(title, tooltip)}
}

type realTrayMenuItem struct {
	item *systray.MenuItem
}

func (i realTrayMenuItem) Clicked() <-chan struct{} {
	return i.item.ClickedCh
}
