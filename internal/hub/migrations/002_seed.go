package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		existing, _ := app.FindFirstRecordByFilter("node_tokens", "token = {:token}", dbx.Params{"token": "test-agent-token"})
		if existing == nil {
			col, _ := app.FindCollectionByNameOrId("node_tokens")
			rec := core.NewRecord(col)
			rec.Set("token", "test-agent-token")
			rec.Set("label", "dev test agent")
			app.Save(rec)
		}

		existingRule, _ := app.FindFirstRecordByFilter("rules", "action = {:action}", dbx.Params{"action": "record"})
		if existingRule == nil {
			rulesCol, _ := app.FindCollectionByNameOrId("rules")
			rule := core.NewRecord(rulesCol)
			rule.Set("node", "*")
			rule.Set("priority", 10)
			rule.Set("action", "record")
			rule.Set("match", `{"host":"*.example.com","path":"/api/*","method":"POST"}`)
			rule.Set("spec", "{}")
			rule.Set("enabled", true)
			app.Save(rule)
		}

		return nil
	}, nil)
}
