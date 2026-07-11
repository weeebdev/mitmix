package hub

import (
	"log"

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

	return pb.Start()
}

func (h *Hub) registerMiddlewares(se *core.ServeEvent) {
}

func (h *Hub) registerRoutes(se *core.ServeEvent) {
	api := se.Router.Group("/api/mitm")

	apiNoAuth := se.Router.Group("/api/mitm")

	apiNoAuth.GET("/agent-connect", h.handleAgentConnect)

	api.POST("/flows", h.handleIngestFlows)
	api.GET("/flows", h.handleListFlows)

	api.GET("/rules", h.handleListRules)
	api.POST("/rules", h.handleCreateRule)

	api.GET("/nodes", h.handleListNodes)
}
