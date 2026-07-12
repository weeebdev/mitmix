package hub

import (
	"log"
	"net/http"
	"os"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/plugins/migratecmd"
	"github.com/pocketbase/pocketbase/tools/osutils"

	_ "github.com/adil/mitm-decentralized/internal/hub/migrations"
)

type Hub struct {
	core.App
	ws *WSManager
}

func New(app core.App) *Hub {
	return &Hub{
		App: app,
		ws:  NewWSManager(nil),
	}
}

func (h *Hub) Start() error {
	pb, ok := h.App.(*pocketbase.PocketBase)
	if !ok {
		return h.App.Bootstrap()
	}

	migratecmd.MustRegister(pb, pb.RootCmd, migratecmd.Config{
		Automigrate: osutils.IsProbablyGoRun(),
	})

	h.ws = NewWSManager(h)

	h.registerRuleHooks()

	pb.OnServe().BindFunc(func(se *core.ServeEvent) error {
		h.registerMiddlewares(se)
		h.registerRoutes(se)
		h.startRetention()
		log.Println("mitm-decentralized hub started")
		return se.Next()
	})

	// Default args for Docker
	args := []string{"serve", "--http=0.0.0.0:8090"}
	if cert := os.Getenv("HUB_TLS_CERT"); cert != "" {
		key := os.Getenv("HUB_TLS_KEY")
		if key != "" {
			args = append(args, "--cert="+cert, "--key="+key)
			if https := os.Getenv("HUB_HTTPS"); https != "" {
				args = append(args, "--https="+https)
			}
		}
	}
	pb.RootCmd.SetArgs(args)

	return pb.Start()
}

func (h *Hub) registerMiddlewares(se *core.ServeEvent) {
}

func (h *Hub) registerRoutes(se *core.ServeEvent) {
	api := se.Router.Group("/api/mitm")

	se.Router.GET("/", func(e *core.RequestEvent) error {
		return e.Redirect(http.StatusFound, "/dashboard")
	})
	se.Router.GET("/ws/agent-connect", h.handleAgentConnect)
	se.Router.POST("/api/mitm/flows", h.handleIngestFlows)

	se.Router.GET("/dashboard/{path...}", h.handleDashboard)
	se.Router.GET("/dashboard", h.handleDashboard)
	se.Router.GET("/ws/dash", h.handleDashboardWS)

	api.GET("/flows", h.handleListFlows)
	api.GET("/flows/{id}", h.handleGetFlow)
	api.GET("/rules", h.handleListRules)
	api.POST("/rules", h.handleCreateRule)
	api.PUT("/rules/{id}", h.handleUpdateRule)
	api.DELETE("/rules/{id}", h.handleDeleteRule)
	api.POST("/rules/reorder", h.handleReorderRules)
	api.GET("/nodes", h.handleListNodes)
	api.GET("/tokens", h.handleListTokens)
	api.POST("/tokens", h.handleCreateToken)
	api.DELETE("/tokens/{id}", h.handleDeleteToken)
	api.GET("/queries", h.handleListQueries)
	api.POST("/queries", h.handleCreateQuery)
	api.DELETE("/queries/{id}", h.handleDeleteQuery)
	api.GET("/queries/{id}/run", h.handleRunQuery)
	se.Router.POST("/mcp", h.handleMCP)
}
