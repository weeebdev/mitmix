package hub

import (
	"embed"
	"io/fs"
	"log"
	"strings"

	"github.com/pocketbase/pocketbase/core"
)

//go:embed site/dist/index.html
//go:embed site/dist/assets/index.css
//go:embed site/dist/assets/index.js
var dashboardFS embed.FS

func (h *Hub) handleDashboard(e *core.RequestEvent) error {
	path := strings.TrimPrefix(e.Request.URL.Path, "/dashboard/")
	if path == "" || !strings.Contains(path, ".") {
		path = "index.html"
	}

	sub, err := fs.Sub(dashboardFS, "site/dist")
	if err != nil {
		log.Printf("dashboard sub error: %v", err)
		return e.InternalServerError("read failed", nil)
	}

	data, err := fs.ReadFile(sub, path)
	if err != nil {
		path = "index.html"
		data, err = fs.ReadFile(sub, path)
		if err != nil {
			return e.NotFoundError("not found", nil)
		}
	}

	contentType := "text/plain"
	if strings.HasSuffix(path, ".css") {
		contentType = "text/css"
	} else if strings.HasSuffix(path, ".js") {
		contentType = "application/javascript"
	} else {
		contentType = "text/html"
	}

	return e.Blob(200, contentType, data)
}
