package main

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/BradDeA/blog-aggregator/internal/config"
	"github.com/BradDeA/blog-aggregator/internal/database"

	_ "github.com/lib/pq"
)

func main() {

	cfg, funcError := config.Read()
	if funcError != nil {
		fmt.Println(funcError)
		os.Exit(1)
	}

	db, e := sql.Open("postgres", cfg.URL)
	if e != nil {
		os.Exit(1)
	}
	defer db.Close()
	dbQueries := database.New(db)

	s := config.State{
		Db:  dbQueries,
		Cfg: cfg,
	}

	c := config.Commands{
		CommandFuncs: make(map[string]func(*config.State, config.Command) error),
	}

	c.Register("login", config.HandlerLogin)
	c.Register("register", config.HandlerRegister)
	c.Register("reset", config.HandlerReset)
	c.Register("users", config.Users)
	c.Register("agg", config.Aggregate)
	c.Register("addfeed", config.AddFeed)
	c.Register("feeds", config.MiddlewareLoggedIn(config.HandlerAddFeed))
	c.Register("follow", config.MiddlewareLoggedIn(config.FollowFeed))
	c.Register("following", config.MiddlewareLoggedIn(config.Following))
	c.Register("unfollow", config.MiddlewareLoggedIn(config.Unfollow))
	c.Register("browse", config.Browse)

	if len(os.Args) < 2 {
		fmt.Println("Not enough arguments")
		os.Exit(1)
	}

	cmdName := os.Args[1]
	args := os.Args[2:]

	cmd := config.Command{
		Name: cmdName,
		Args: args,
	}

	err := c.Run(&s, cmd)
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

}
