package main

import (
	"gobattle/common/api"
	"gobattle/game/config"
)

var roleSetups = map[string]func(a *app){
	api.GroupLogic: setupLogic,
	api.GroupMatch: setupMatch,
}

func main() {
	config.Load()
	cfg := config.Game
	setup, ok := roleSetups[cfg.Role]
	if !ok {
		panic("unknown role: " + cfg.Role)
	}
	app := newApp(cfg)
	setup(app)
	app.run()
}
