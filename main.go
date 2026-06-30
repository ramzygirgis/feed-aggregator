package main


import (
	"fmt"
	"os"
	"database/sql"
	_ "github.com/lib/pq"
	"github.com/ramzygirgis/feed-aggregator/internal/config"
	"github.com/ramzygirgis/feed-aggregator/internal/database"
)



func main() {
	cfg, err := config.Read()
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	db, err := sql.Open("postgres", cfg.DBURL)
	if err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}

	dbQueries := database.New(db)

	s := state{cfg: &cfg, db: dbQueries}
	c := InitializeCommandMap()
	c.register("login", handlerLogin)
	c.register("register", handlerRegister)
	c.register("reset", handlerReset)
	c.register("users", handlerUsers)
	c.register("agg", handlerAgg)
	c.register("addfeed", handlerAddfeed)
	c.register("feeds", handlerFeeds)
	c.register("follow", handlerFollow)

	if len(os.Args) < 2 {
		fmt.Printf("no command name passed\n")
		os.Exit(1)
	}
	name := os.Args[1]
	args := os.Args[2:]
	cmd := command{name: name, args: args}
	
	if err = c.run(&s, cmd); err != nil {
		fmt.Printf("%s\n", err)
		os.Exit(1)
	}
}
