package resident

import (
	"context"
	"errors"
	"fmt"
)

var ErrInvalidHost = errors.New("invalid resident host")

type Host interface {
	Menu() []MenuItem
	OpenPortcut(context.Context) error
	Quit(context.Context) error
}

type Controller struct {
	menu    []MenuItem
	session Session
}

func NewHost(session Session) (*Controller, error) {
	controller := &Controller{
		menu:    DefaultMenu(),
		session: session,
	}

	if err := controller.Validate(); err != nil {
		return nil, err
	}

	return controller, nil
}

func (c *Controller) Validate() error {
	if c.session == nil {
		return fmt.Errorf("%w: session is required", ErrInvalidHost)
	}
	if err := ValidateMenu(c.menu); err != nil {
		return err
	}

	return nil
}

func (c *Controller) Menu() []MenuItem {
	menu := make([]MenuItem, len(c.menu))
	copy(menu, c.menu)
	return menu
}

func (c *Controller) OpenPortcut(ctx context.Context) error {
	return c.session.Open(ctx, openRequestFor(c.session.Snapshot()))
}

func (c *Controller) Quit(ctx context.Context) error {
	return c.session.Shutdown(ctx, ShutdownRequest{Wait: true})
}

func openRequestFor(snapshot SessionSnapshot) OpenRequest {
	request := OpenRequest{Reason: OpenReasonLaunch}
	if snapshot.Active {
		request.Reason = OpenReasonReopen
	}

	return request
}
