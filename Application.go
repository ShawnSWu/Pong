package main

import (
	"Pong/core"
	"Pong/logger"
)

func main() {
	logger.Log.Init()
	core.Start()
}
