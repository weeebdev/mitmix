package migrations

import (
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		settings := app.Settings()
		settings.Meta.AppName = "mitm-decentralized"
		settings.Meta.HideControls = true
		app.Save(settings)

		email := os.Getenv("HUB_ADMIN_EMAIL")
		pass := os.Getenv("HUB_ADMIN_PASSWORD")
		if email == "" || pass == "" {
			return nil
		}

		existing, _ := app.FindAuthRecordByEmail(core.CollectionNameSuperusers, email)
		if existing != nil {
			return nil
		}

		col, err := app.FindCollectionByNameOrId(core.CollectionNameSuperusers)
		if err != nil {
			return err
		}

		rec := core.NewRecord(col)
		rec.Set("email", email)
		rec.Set("password", pass)
		rec.Set("passwordConfirm", pass)
		return app.Save(rec)
	}, nil)
}
