package main

import (
	"log"

	"github.com/pocketbase/pocketbase"

	"github.com/adil/mitm-decentralized/internal/hub"
)

func main() {
	app := pocketbase.NewWithConfig(pocketbase.Config{
		DefaultDataDir: "/pb_data",
	})
	h := hub.New(app)
	if err := h.Start(); err != nil {
		log.Fatal(err)
	}
}
