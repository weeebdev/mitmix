package migrations

import (
	"crypto/rand"
	"encoding/hex"
	"log"
	"os"

	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		var count int
		app.DB().Select("count(*)").From("_superusers").Row(&count)
		if count > 0 {
			return nil
		}
		email := os.Getenv("HUB_ADMIN_EMAIL")
		pass := os.Getenv("HUB_ADMIN_PASSWORD")
		if email == "" {
			email = "admin@mitm.local"
		}
		if pass == "" {
			b := make([]byte, 24)
			rand.Read(b)
			pass = hex.EncodeToString(b)
			log.Printf("generated random admin password (set HUB_ADMIN_PASSWORD to override): %s", pass)
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
