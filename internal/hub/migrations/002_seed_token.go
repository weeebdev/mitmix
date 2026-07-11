package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		col, err := app.FindCollectionByNameOrId("node_tokens")
		if err != nil {
			return err
		}

		existing, _ := app.FindFirstRecordByFilter("node_tokens", "token = {:token}", dbx.Params{"token": "test-agent-token"})
		if existing != nil {
			return nil
		}

		rec := core.NewRecord(col)
		rec.Set("token", "test-agent-token")
		rec.Set("label", "dev test agent")
		return app.Save(rec)
	}, nil)
}
