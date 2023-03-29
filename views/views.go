// Package views contains the views for the application.
package views

import (
	"fmt"

	"github.com/labstack/echo/v5"
	"github.com/nmcapule/oclz-go/syncer"
	"github.com/nmcapule/oclz-go/views/authentication"
	"github.com/pocketbase/pocketbase"
)

type hooker interface {
	Hook(*echo.Group) error
}

// RootView is the root view for the application.
type RootView struct {
	App    *pocketbase.PocketBase
	Syncer *syncer.Syncer
}

// Hook hooks up the echo HTTP router with the defined views.
func (r *RootView) Hook(e *echo.Echo) error {
	root := e.Group("")

	modules := []hooker{
		&authentication.View{
			App:         r.App,
			Syncer:      r.Syncer,
			GroupPrefix: "/authentication",
		},
	}
	for _, m := range modules {
		if err := m.Hook(root); err != nil {
			return fmt.Errorf("hooking module: %w", err)
		}
	}
	return nil
}
