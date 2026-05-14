package main

import (
	"fmt"
	"github.com/ramzygirgis/feed-aggregator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		return
	}

	err = cfg.SetUser("ramzy")
	if err != nil {
		return
	}
	cfg, err = config.Read()
	fmt.Printf("DBURL: %s, CurrentUserName: %s\n", cfg.DBURL, cfg.CurrentUserName)	
}
