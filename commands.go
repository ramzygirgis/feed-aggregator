package main


import (
	"fmt"
	"time"
	"context"
  "github.com/ramzygirgis/feed-aggregator/internal/config"
	"github.com/ramzygirgis/feed-aggregator/internal/database"
	"github.com/google/uuid"
)


type state struct {
	db *database.Queries
	cfg *config.Config
}


type command struct {
	name string
	args []string
}


type commands struct {
	all map[string]func(*state, command) error
}


// helpers


func InitializeCommandMap() commands {
	all := make(map[string]func(*state, command) error)
	return commands{all: all}
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


// handlers


func handlerLogin(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("no arguments provided for the login command\n")
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments provided for the login command; 1 expected, %d given\n", len(cmd.args))
	}
	username := cmd.args[0]
	
	_, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return err
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}
	fmt.Printf("Success! Username has been set to %s.\n", username)
	return nil
}


func handlerRegister(s *state, cmd command) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("no arguments provided for the register command\n")
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments provided for the register command; 1 expected, %d given\n", len(cmd.args))
	}

	username := cmd.args[0]
	uuid := uuid.New()
	createdAt := time.Now()
	updatedAt := createdAt

	params := database.CreateUserParams{
		ID: uuid,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Name: username,
	}

	_, err := s.db.CreateUser(context.Background(), params)
	if err != nil {
		return err
	}

	err = s.cfg.SetUser(username)
	if err != nil {
		return err
	}
	
	fmt.Printf("Success! New user %s has been registered.\n", username)
	return nil
}


func handlerReset(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("too many arguments provided for the register command; 0 expected, %d given\n", len(cmd.args))
	}
	
	err := s.db.ResetDb(context.Background())
	if err != nil {
		return err
	}

	fmt.Printf("Database has been successfully reset.\n")
	return nil
}


func handlerUsers(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("too many arguments provided for the users command; 0 expected, %d given\n", len(cmd.args))
	}

	items, err := s.db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, name := range items {
		if name == s.cfg.CurrentUserName {
			fmt.Printf("* %s (current)\n", name)
		} else {
			fmt.Printf("* %s\n", name)
		}
	}

	return nil
}


func handlerAgg(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("too many arguments provided for the agg command; 0 expected, %d given\n", len(cmd.args))
	}

	r, err := fetchFeed(context.Background(), "https://www.wagslane.dev/index.xml")
	if err != nil {
		return err
	}

	fmt.Printf("%+v\n", *r)
	return nil
}


func handlerAddfeed(s *state, cmd command) error {
	if len(cmd.args) < 2 {
		return fmt.Errorf("not enough arguments provided for the addfeed command; 2 expected, %d given\n", len(cmd.args))
	}
	if len(cmd.args) > 2 {
		return fmt.Errorf("too many arguments provided for the addfeed command; 2 expected, %d given\n", len(cmd.args))
	}

	username := s.cfg.CurrentUserName 
	User, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return err
	}
	userid := User.ID

	t := time.Now()

	params := database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		Name: cmd.args[0],
		Url: cmd.args[1],
		UserID: userid,
	}

	_, err = s.db.CreateFeed(context.Background(), params)
	if err != nil {
		return err
	}
	
	fmt.Printf("Feed Name: %s\n", cmd.args[0])
	fmt.Printf("Url: %s\n", cmd.args[1])
	fmt.Printf("UserID: %s\n", userid)


	return nil
}


func handlerFeeds(s *state, cmd command) error {
	if len(cmd.args) > 0 {
		return fmt.Errorf("too many arguments provided for the feeds command; 0 expected, %d given\n", len(cmd.args))
	}

	feeds, err := s.db.GetFeeds(context.Background())
	if err != nil {
		return err
	}

	var user database.User
	for i := 0; i < len(feeds); i++ {
		fmt.Printf("********* FEED %d *********\n", i + 1)
		fmt.Printf("Name: %s\n", feeds[i].Name)
		fmt.Printf("Url: %s\n", feeds[i].Url)
		user, err = s.db.GetUserById(context.Background(), feeds[i].UserID)
		if err != nil {
			return err
		}
		fmt.Printf("UserName: %s\n", user.Name)
	}
	fmt.Println("********************")
	
	return nil
}


func handlerFollow(s *state, cmd command) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("not enough arguments provided for the follow command; 1 expected, %d given\n", len(cmd.args))
	}
	if len(cmd.args) > 2 {
		return fmt.Errorf("too many arguments provided for the follow command; 1 expected, %d given\n", len(cmd.args))
	}
	feedURL := cmd.args[0]
	username := s.cfg.CurrentUserName
	Feed, err := s.db.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return err
	}

	User, err := s.db.GetUser(context.Background(), username)
	if err != nil {
		return err
	}

	t := time.Now()
	params := database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: t, UpdatedAt: t, UserID: User.ID, FeedID: Feed.ID}

	_, err = s.db.CreateFeedFollow(context.Background(), params) // hit NewRow with an _ if not useful
	if err != nil {
		return err
	}
	fmt.Printf("********* FEED FOLLOW *********\n")
	fmt.Printf("Feed Name: %s\n", Feed.Name)
	fmt.Printf("Username: %s\n", username)
	
	return nil
}


func handlerFollowing(s *state, cmd command) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("too many arguments provided for the following command; 0 expected, %d given\n", len(cmd.args))
	}
	currentUser, err := s.db.GetUser(context.Background(), s.cfg.CurrentUserName)
	if err != nil {
		return err
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), currentUser.ID)
	if err != nil {
		return err
	}
	
	fmt.Printf("****** FEED FOLLOWS FOR %s ******\n", s.cfg.CurrentUserName)

	if len(follows) == 0 {
		fmt.Printf("%s follows no feeds.\n", s.cfg.CurrentUserName)
		return nil
	}
	var cur_feed database.Feed
	for i := 0; i < len(follows); i++ {
		cur_feed, err = s.db.GetFeed(context.Background(), follows[i].FeedID)
		fmt.Printf("- %s\n", cur_feed.Name)
	}
	return nil

}
