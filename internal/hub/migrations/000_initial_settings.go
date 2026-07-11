package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		settings := app.Settings()
		settings.Meta.AppName = "mitm-decentralized"
		settings.Meta.HideControls = true
		app.Save(settings)
		return nil
	}, nil)
}
