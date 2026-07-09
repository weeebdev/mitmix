package main

import (
	"log"

	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{})

	app.OnServe().BindFunc(func(se *core.ServeEvent) error {
		// TODO: register /api/mitm/agent-connect (agent WebSocket, internal/ws/agent_ws.go)
		// TODO: register /api/mitm/flows (batch flow ingest, internal/api/routes.go)
		log.Println("mitm-decentralized hub starting")
		return se.Next()
	})

	if err := app.Start(); err != nil {
		log.Fatal(err)
	}
}
