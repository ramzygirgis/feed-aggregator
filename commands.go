package main

import(
	"fmt"
  "github.com/ramzygirgis/feed-aggregator/internal/config"
)


type state struct {
	cfg *config.Config
}


type command struct {
	name string
	args []string
}


type commands struct {
	all map[string]func(*state, command) error
}


func InitializeCommandMap() commands {
	all := make(map[string]func(*state, command) error)
	return commands{all: all}
}


func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("no arguments provided for the log command\n")
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments provided for the login command; 1 expected, %d given\n", len(cmd.args))
	}
	username := cmd.args[0]
	err := s.cfg.SetUser(username)
	if err != nil {
		return err
	}
	fmt.Printf("Success! Username has been set to %s.\n", username)
	return nil
}


func (c *commands) run(s *state, cmd command) error {
	callback, ok := c.all[cmd.name]
	if !ok {
		return fmt.Errorf("command name '%s' not found\n", cmd.name)
	}

	err := callback(s, cmd)
	if err != nil {
		return err
	}
	return nil
}


func (c *commands) register(name string, f func(*state, command) error) {
	c.all[name] = f
}



