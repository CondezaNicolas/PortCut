package resident

import (
	"errors"
	"fmt"
)

var ErrInvalidMenu = errors.New("invalid resident menu")

type MenuAction string

const (
	MenuActionOpen MenuAction = "open"
	MenuActionQuit MenuAction = "quit"
)

type MenuItem struct {
	Label  string
	Action MenuAction
}

func DefaultMenu() []MenuItem {
	return []MenuItem{
		{Label: "Open Portcut", Action: MenuActionOpen},
		{Label: "Quit", Action: MenuActionQuit},
	}
}

func ValidateMenu(menu []MenuItem) error {
	if len(menu) != 2 {
		return fmt.Errorf("%w: expected exactly 2 items", ErrInvalidMenu)
	}
	if menu[0] != (MenuItem{Label: "Open Portcut", Action: MenuActionOpen}) {
		return fmt.Errorf("%w: first item must be Open Portcut", ErrInvalidMenu)
	}
	if menu[1] != (MenuItem{Label: "Quit", Action: MenuActionQuit}) {
		return fmt.Errorf("%w: second item must be Quit", ErrInvalidMenu)
	}

	return nil
}
