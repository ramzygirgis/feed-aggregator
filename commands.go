package main


import (
	"fmt"
	"time"
	"context"
  "database/sql"
	"strings"
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


func scrapeFeeds(s *state) error {
	nextFeed, err := s.db.GetNextFeedToFetch(context.Background())
	if err != nil {
		return err
	}

	err = s.db.MarkFeedFetched(context.Background(), nextFeed.ID)
	if err != nil {
		return err
	}

	RSSData, err := fetchFeed(context.Background(), nextFeed.Url)
	items := RSSData.Channel.Item
	var t time.Time
	var params database.CreatePostParams
	var nullDesc sql.NullString
	var nullTime sql.NullTime
	for i := 0; i < len(items); i++ {

		if items[i].Link == "" {
			continue
		}

		t, err = time.Parse(time.RFC1123Z, items[i].PubDate)
		if err != nil {
			t, err = time.Parse(time.RFC3339, items[i].PubDate)
			if err != nil{
				fmt.Printf("%s\n", err)
				t = time.Time{}
			}
		}

		nullDesc = sql.NullString{
			String: items[i].Description,
			Valid:  (items[i].Description != ""),
		}

		nullTime = sql.NullTime{
			Time: t,
			Valid: (t != time.Time{}),
		}

		params = database.CreatePostParams{
			ID: uuid.New(),
			CreatedAt: nextFeed.CreatedAt,
			UpdatedAt: nextFeed.UpdatedAt,
			Title: items[i].Title,
			Url: items[i].Link,
			Description: nullDesc,
			PublishedAt: nullTime,
			FeedID: nextFeed.ID,
		}

		_, err = s.db.CreatePost(context.Background(), params)
		if err != nil {
			if strings.Contains(err.Error(), "unique") {
				continue
			}
			fmt.Printf("%s\n", err.Error())
			return err
		}	

		fmt.Printf("***** Item %d *****\n", i+1)
		fmt.Printf("%s\n", items[i].Title)
	}

	return nil
}


func handlerAgg(s *state, cmd command) error {
	timeBetweenRequests, err := time.ParseDuration("300ms")
	if err != nil {
		return err
	}
	if len(cmd.args) == 1 {
		timeBetweenRequests, err = time.ParseDuration(cmd.args[0])
		if err != nil {
			return err
		}
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments provided for the agg command; 0/1 expected, %d given\n", len(cmd.args))
	}

	duration_string := timeBetweenRequests.String()
	fmt.Printf("Collecting feeds every %s\n", duration_string)

	ticker := time.NewTicker(timeBetweenRequests)
	for ; ; <-ticker.C {
		err = scrapeFeeds(s)
		if err != nil {
			return err
		}
	}
	return nil
}


func handlerAddfeed(s *state, cmd command, User database.User) error {
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

	t := time.Now()

	feedParams := database.CreateFeedParams{
		ID: uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		Name: cmd.args[0],
		Url: cmd.args[1],
		UserID: User.ID,
	}

	_, err = s.db.CreateFeed(context.Background(), feedParams)
	if err != nil {
		return err
	}
	
	feedFollowParams := database.CreateFeedFollowParams{
		ID: uuid.New(),
		CreatedAt: t,
		UpdatedAt: t,
		UserID: User.ID,
		FeedID: feedParams.ID,
	}
	_, err = s.db.CreateFeedFollow(context.Background(), feedFollowParams)
	if err != nil {
		return err
	}

	fmt.Printf("Feed Name: %s\n", cmd.args[0])
	fmt.Printf("Url: %s\n", cmd.args[1])
	fmt.Printf("UserID: %s\n", User.ID)

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


func handlerFollow(s *state, cmd command, User database.User) error {
	if len(cmd.args) < 1 {
		return fmt.Errorf("not enough arguments provided for the follow command; 1 expected, %d given\n", len(cmd.args))
	}
	if len(cmd.args) > 2 {
		return fmt.Errorf("too many arguments provided for the follow command; 1 expected, %d given\n", len(cmd.args))
	}
	feedURL := cmd.args[0]
	username := User.Name
	Feed, err := s.db.GetFeedByURL(context.Background(), feedURL)
	if err != nil {
		return err
	}


	t := time.Now()
	params := database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: t, UpdatedAt: t, UserID: User.ID, FeedID: Feed.ID}

	_, err = s.db.CreateFeedFollow(context.Background(), params)
	if err != nil {
		return err
	}
	fmt.Printf("********* FEED FOLLOW *********\n")
	fmt.Printf("Feed Name: %s\n", Feed.Name)
	fmt.Printf("Username: %s\n", username)
	
	return nil
}


func handlerFollowing(s *state, cmd command, User database.User) error {
	if len(cmd.args) != 0 {
		return fmt.Errorf("too many arguments provided for the following command; 0 expected, %d given\n", len(cmd.args))
	}

	follows, err := s.db.GetFeedFollowsForUser(context.Background(), User.ID)
	if err != nil {
		return err
	}
	
	fmt.Printf("****** FEED FOLLOWS FOR %s ******\n", User.Name)

	if len(follows) == 0 {
		fmt.Printf("%s follows no feeds.\n", User.Name)
		return nil
	}
	var cur_feed database.Feed
	for i := 0; i < len(follows); i++ {
		cur_feed, err = s.db.GetFeed(context.Background(), follows[i].FeedID)
		fmt.Printf("- %s\n", cur_feed.Name)
	}
	return nil
}


func handlerUnfollow(s *state, cmd command, User database.User) error {
	if len(cmd.args) == 0 {
		return fmt.Errorf("not enough arguments provided for the unfollow command; 1 expected, 0 given\n")
	}
	if len(cmd.args) > 1 {
		return fmt.Errorf("too many arguments provided for the unfollow command; 1 expected, %d given\n", len(cmd.args))
	}
	
	url := cmd.args[0]
	Feed, err := s.db.GetFeedByURL(context.Background(), url)
	if err != nil {
		return err
	}

	params := database.DeleteFeedFollowParams{
		UserID: User.ID,
		FeedID: Feed.ID,
	}
	err = s.db.DeleteFeedFollow(context.Background(), params)
	if err != nil {
		return err
	}
	fmt.Printf("%s has succesfully unfollowed %s\n", User.Name, Feed.Name)
	return nil
}
