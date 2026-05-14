package main

import (
	"fmt"
	"os"
	"github.com/ramzygirgis/feed-aggregator/internal/config"
)

func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}

	s := state{cfg: &cfg}
	c := InitializeCommandMap()
	c.register("login", handlerLogin)

	if len(os.Args) < 2 {
		fmt.Printf("no command name passed\n")
		os.Exit(1)
	}
	name := os.Args[1]
	args := os.Args[2:]
	cmd := command{name: name, args: args}
	
	if err = c.run(&s, cmd); err != nil {
		fmt.Printf("%s", err)
		os.Exit(1)
	}
}
