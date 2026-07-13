package migrations

import (
	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase/core"
	m "github.com/pocketbase/pocketbase/migrations"
)

func init() {
	m.Register(func(app core.App) error {
		var count int
		app.DB().Select("count(*)").From("pragma_table_info('flows')").Where("name = {:name}", dbx.Params{"name": "app_name"}).Row(&count)
		if count == 0 {
			app.DB().NewQuery("ALTER TABLE flows ADD COLUMN app_name TEXT DEFAULT ''").Execute()
		}
		app.DB().Select("count(*)").From("pragma_table_info('flows')").Where("name = {:name}", dbx.Params{"name": "source_host"}).Row(&count)
		if count == 0 {
			app.DB().NewQuery("ALTER TABLE flows ADD COLUMN source_host TEXT DEFAULT ''").Execute()
		}
		return nil
	}, func(app core.App) error {
		return nil
	})
}
