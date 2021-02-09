package main

import (
	"log"

	"github.com/vishal1132/servicedesk/config"
)

func main() {
	cfg, err := config.LoadEnv()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}
	logger := config.DefaultLogger(cfg)
	if err = runServer(cfg, logger); err != nil {
		log.Fatalf("failed to run new consumer server: %v", err.Error())
	}
}
