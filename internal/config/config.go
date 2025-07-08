package config

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/BradDeA/blog-aggregator/internal/database"
	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Config struct {
	URL         string `json:"db_url"`
	CurrentUser string `json:"current_user_name"`
}

type State struct {
	Db  *database.Queries
	Cfg *Config
}

type Command struct {
	Name string
	Args []string
}

type Commands struct {
	CommandFuncs map[string]func(*State, Command) error
}

type RSSFeed struct {
	Channel struct {
		Title       string    `xml:"title"`
		Link        string    `xml:"link"`
		Description string    `xml:"description"`
		Item        []RSSItem `xml:"item"`
	} `xml:"channel"`
}

type RSSItem struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
}

func Read() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	var config *Config
	content, err := os.ReadFile(home + "/.gatorconfig.json")
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(content)
	decoder := json.NewDecoder(reader)
	e := decoder.Decode(&config)
	if e != nil {
		return nil, e
	}
	return config, nil
}

func (c *Config) SetUser(s string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	jsonFile, err := os.OpenFile(home+"/.gatorconfig.json", os.O_WRONLY|os.O_TRUNC, 644)
	if err != nil {
		return err
	}
	defer jsonFile.Close()

	encoder := json.NewEncoder(jsonFile)
	c.CurrentUser = s
	encodeError := encoder.Encode(c)
	if encodeError != nil {
		return encodeError
	}

	return nil
}

func HandlerLogin(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return errors.New("usage: gator login <username>")
	}

	_, e := s.Db.GetUser(context.Background(), cmd.Args[0])
	if e != nil {
		return fmt.Errorf("couldn't find user: %w", e)
	}

	err := s.Cfg.SetUser(cmd.Args[0])
	if err != nil {
		return err
	}

	fmt.Println("User has been set")
	return nil
}

func (c *Commands) Run(s *State, cmd Command) error {
	name, bl := c.CommandFuncs[cmd.Name]
	if !bl {
		return errors.New("command not found")
	}
	return name(s, cmd)
}

func (c *Commands) Register(name string, f func(*State, Command) error) {
	c.CommandFuncs[name] = f
}

func HandlerRegister(s *State, cmd Command) error {
	if len(cmd.Args) != 1 {
		return errors.New("usage: gator register <username>")
	}

	val, err := s.Db.CreateUser(context.Background(), database.CreateUserParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), Name: cmd.Args[0]})
	if err != nil {
		os.Exit(1)
	}
	s.Cfg.SetUser(val.Name)
	fmt.Println("User Created", val.Name)
	return nil
}

func HandlerReset(s *State, cmd Command) error {

	err := s.Db.DeleteUsers(context.Background())
	if err != nil {
		os.Exit(1)
	}
	return nil
}

func Users(s *State, cmd Command) error {
	names, err := s.Db.GetUsers(context.Background())
	if err != nil {
		return err
	}

	for _, username := range names {
		if username == s.Cfg.CurrentUser {
			fmt.Println(username + " (current)")
		} else {
			fmt.Println(username)
		}

	}

	return nil
}

func fetchFeed(ctx context.Context, feedURL string) (*RSSFeed, error) {

	RSS := &RSSFeed{}
	client := http.Client{}

	req, err := http.NewRequestWithContext(ctx, "GET", feedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "gator")

	feed, e := client.Do(req)
	if e != nil {
		return nil, e
	}
	defer feed.Body.Close()

	xmldecoder := xml.NewDecoder(feed.Body)

	xmlerr := xmldecoder.Decode(RSS)
	if xmlerr != nil {
		return nil, xmlerr
	}

	for i := range RSS.Channel.Item {
		RSS.Channel.Item[i].Title = html.UnescapeString(RSS.Channel.Item[i].Title)
		RSS.Channel.Item[i].Description = html.UnescapeString(RSS.Channel.Item[i].Description)
	}
	RSS.Channel.Title = html.UnescapeString(RSS.Channel.Title)
	RSS.Channel.Description = html.UnescapeString(RSS.Channel.Description)

	return RSS, nil
}

func Aggregate(s *State, cmd Command) error {
	if len(cmd.Args) == 0 {
		fmt.Println("time between request required")
		os.Exit(1)
	}
	time_between_reqs := cmd.Args[0]
	timeVal, timeErr := time.ParseDuration(time_between_reqs)
	if timeErr != nil {
		return timeErr
	}
	fmt.Println("Printing feeds every " + time_between_reqs)

	ticker := time.NewTicker(timeVal)
	for ; ; <-ticker.C {
		scrapeFeeds(s)
	}

}

func AddFeed(s *State, cmd Command) error {
	if len(cmd.Args) < 2 {
		os.Exit(1)
	}

	feedname := cmd.Args[0]
	feedurl := cmd.Args[1]
	feedId := uuid.New()

	user, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUser)
	if err != nil {
		return err
	}
	newfeed, feederr := s.Db.CreateFeed(context.Background(), database.CreateFeedParams{ID: feedId, CreatedAt: time.Now(), UpdatedAt: time.Now(), UserID: user.ID, Name: feedname, Url: feedurl})
	if feederr != nil {
		return feederr
	}
	_, followerr := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), UserID: user.ID, FeedID: newfeed.ID})
	if followerr != nil {
		return followerr
	}

	fmt.Println("Feed ID:", newfeed.ID, "Feed Name:", newfeed.Name, "Feed URL:", newfeed.Url)
	return nil
}

func HandlerAddFeed(s *State, cmd Command, user database.User) error {

	feedInfo, userserr := s.Db.JoinFeedsTable(context.Background())
	if userserr != nil {
		return userserr
	}

	for i := range feedInfo {
		unpackInfo := feedInfo[i]
		fmt.Println(unpackInfo.Name, "\n", unpackInfo.Url, "\n", unpackInfo.Name_2)
	}
	return nil
}

func FollowFeed(s *State, cmd Command, user database.User) error {

	user, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUser)
	if err != nil {
		return err
	}

	feedURL := cmd.Args[0]
	feed, err := s.Db.FeedLookup(context.Background(), feedURL)
	if err != nil {
		return err
	}
	follow, followerror := s.Db.CreateFeedFollow(context.Background(), database.CreateFeedFollowParams{ID: uuid.New(), CreatedAt: time.Now(), UpdatedAt: time.Now(), UserID: user.ID, FeedID: feed.ID})
	if followerror != nil {
		return followerror
	}

	fmt.Println(follow.UserName, "is now following", follow.FeedName)
	return nil
}

func Following(s *State, cmd Command, user database.User) error {

	user, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUser)
	if err != nil {
		return err
	}

	fol, err := s.Db.GetFeedFollowsForUser(context.Background(), user.ID)
	if err != nil {
		return err
	}

	fmt.Println(s.Cfg.CurrentUser)
	for i := range fol {
		fmt.Println(fol[i].FeedName)
	}
	return nil
}

func MiddlewareLoggedIn(handler func(s *State, cmd Command, user database.User) error) func(*State, Command) error {
	return func(s *State, cmd Command) error {
		user, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUser)
		if err != nil {
			return err
		}
		return handler(s, cmd, user)
	}
}

func Unfollow(s *State, cmd Command, user database.User) error {
	feedURL := cmd.Args[0]

	user, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUser)
	if err != nil {
		return err
	}

	feed, err := s.Db.FeedLookup(context.Background(), feedURL)
	if err != nil {
		return err
	}

	finerr := s.Db.UnfollowFeed(context.Background(), database.UnfollowFeedParams{UserID: user.ID, FeedID: feed.ID})
	if finerr != nil {
		return finerr
	}
	return nil
}

func scrapeFeeds(s *State) error {
	feed, errGet := s.Db.GetNextFeedToFetch(context.Background())
	if errGet != nil {
		return errGet
	}

	feedFound, errFeed := s.Db.FeedLookup(context.Background(), feed)
	if errFeed != nil {
		return errFeed
	}

	feedFetch, errFetch := fetchFeed(context.Background(), feed)
	if errFetch != nil {
		return errFetch
	}

	errMark := s.Db.MarkFeedFetched(context.Background(), feedFound.ID)
	if errMark != nil {
		return errMark
	}

	for _, item := range feedFetch.Channel.Item {

		var parsedTime time.Time
		var errValid error

		layouts := []string{time.RFC1123Z, time.RFC1123, time.RFC822Z, time.RFC822, time.RFC3339}
		for _, layout := range layouts {
			parsedTime, errValid = time.Parse(layout, item.PubDate)
			if errValid == nil {
				break
			}
		}

		if errValid != nil {
			fmt.Printf("FAILED TO PARSE: %s, error: %v\n", item.PubDate, errValid)
		} else {
			fmt.Printf("PARSED: %s -> %s\n", item.PubDate, parsedTime.Format(time.RFC1123Z))
		}

		publishedAt := sql.NullTime{Valid: errValid == nil}
		if errValid == nil {
			publishedAt.Time = parsedTime
		}

		_, err := s.Db.CreatePost(context.Background(), database.CreatePostParams{
			ID:          uuid.New(),
			CreatedAt:   time.Now(),
			UpdatedAt:   time.Now(),
			Title:       sql.NullString{String: item.Title, Valid: item.Title != ""},
			Url:         sql.NullString{String: item.Link, Valid: item.Link != ""},
			Description: sql.NullString{String: item.Description, Valid: item.Description != ""},
			PublishedAt: publishedAt,
			FeedID:      feedFound.ID,
		})
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			continue
		} else if err != nil {
			return err
		}
	}

	return nil
}

func Browse(s *State, cmd Command) error {
	user, err := s.Db.GetUser(context.Background(), s.Cfg.CurrentUser)
	if err != nil {
		return err
	}

	if len(cmd.Args) != 0 {
		limit, convErr := strconv.ParseInt(cmd.Args[0], 10, 32)
		if convErr != nil {
			return convErr
		}
		posts, valueErr := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{UserID: user.ID, Limit: int32(limit)})
		if valueErr != nil {
			return valueErr
		}
		for _, post := range posts {
			if post.PublishedAt.Valid {
				fmt.Printf("%s\n%s\n%s\n%s\n\n", post.PublishedAt.Time.Format(time.RFC1123Z), post.Url.String, post.Title.String, post.Description.String)
			} else {
				fmt.Println("No published date")
			}
		}

	} else {
		limit := 2
		posts, valueErr := s.Db.GetPostsForUser(context.Background(), database.GetPostsForUserParams{UserID: user.ID, Limit: int32(limit)})
		if valueErr != nil {
			return valueErr
		}
		for _, post := range posts {
			if post.PublishedAt.Valid {
				fmt.Printf("%s\n%s\n%s\n%s\n\n", post.PublishedAt.Time.Format(time.RFC1123Z), post.Url.String, post.Title.String, post.Description.String)
			} else {
				fmt.Println("No published date")
			}

		}
	}
	return nil
}
