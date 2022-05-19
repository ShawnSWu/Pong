package main

import (
	"Pong/core"
	"Pong/logger"
)

func main() {
	logger.Log.Init()
	logger.Log.Info("Server launching..")
	core.StartService()
}
