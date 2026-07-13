package migrations

import (
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		exists := false
		app.DB().NewQuery("SELECT COUNT(*) FROM pragma_table_info('flows') WHERE name='app_name'").Row(&exists)
		if !exists {
			app.DB().NewQuery("ALTER TABLE flows ADD COLUMN app_name TEXT DEFAULT ''").Execute()
		}
		app.DB().NewQuery("SELECT COUNT(*) FROM pragma_table_info('flows') WHERE name='source_host'").Row(&exists)
		if !exists {
			app.DB().NewQuery("ALTER TABLE flows ADD COLUMN source_host TEXT DEFAULT ''").Execute()
		}
		return nil
	}, func(app core.App) error {
		return nil
	})
}
