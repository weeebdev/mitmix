package hub

import (
	"log"
	"net/http"

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
		log.Println("mitm-decentralized hub started")
		return se.Next()
	})

	// Default args for Docker
	pb.RootCmd.SetArgs([]string{"serve", "--http=0.0.0.0:8090"})

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
	se.Router.POST("/mcp", h.handleMCP)
}
