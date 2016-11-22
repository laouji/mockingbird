package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/mrjones/oauth"
)

const timeLayout = "Mon Jan 02 15:04:05 +0000 2006"

type ConfData struct {
	SearchTerm     string `yaml:"search_term"`
	ConsumerKey    string `yaml:"consumer_key"`
	ConsumerSecret string `yaml:"consumer_secret"`
	AccessToken    string `yaml:"access_token"`
	TokenSecret    string `yaml:"token_secret"`
}

type JsonTime struct {
	time.Time
}

func (t *JsonTime) UnmarshalJSON(b []byte) error {
	s := strings.Trim(string(b), `"`)
	tt, err := time.Parse(timeLayout, s)
	if err != nil {
		return err
	}
	t.Time = tt
	return nil
}

type SearchResults struct {
	Statuses []Status `json:"statuses"`
}

type Status struct {
	CreatedAt JsonTime        `json:"created_at"`
	Id        int64           `json:"id"`
	Entities  TwitterEntities `json:"entities"`
}

type TwitterEntities struct {
	Urls []TwitterUrlEntity `json:"urls"`
}

type TwitterUrlEntity struct {
	ExpandedUrl string `json:"expanded_url"`
}

var (
	configFile = flag.String("c", "config.yml", "location of config file")
)

func main() {
	flag.Parse()
	config := LoadConfig()

	c := oauth.NewConsumer(
		config.ConsumerKey,
		config.ConsumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   "https://api.twitter.com/oauth/request_token",
			AuthorizeTokenUrl: "https://api.twitter.com/oauth/authorize",
			AccessTokenUrl:    "https://api.twitter.com/oauth/access_token",
		})

	accessToken := &oauth.AccessToken{
		Token:  config.AccessToken,
		Secret: config.TokenSecret,
	}

	client, err := c.MakeHttpClient(accessToken)
	if err != nil {
		log.Fatal(err)
	}

	// search using search_term specified in config file
	rawResults, err := search(client, config.SearchTerm)
	if err != nil {
		log.Fatal(err)
	}

	var searchResults SearchResults
	err = json.Unmarshal(rawResults, &searchResults)
	if err != nil {
		log.Fatal(err)
	}

	cutoffTime := time.Now().UTC().Truncate(time.Hour)
	fmt.Printf("retweeting statuses newer than: %s\n", cutoffTime.String())

	// iterate through results and retweet anything within the hour
	for _, result := range searchResults.Statuses {

		// avoid retweeting old material
		if !result.CreatedAt.Truncate(time.Hour).Equal(cutoffTime) {
			continue
		}

		err = retweet(client, result.Id)
		if err != nil {
			fmt.Printf("can't retweet status with id: %d, %s\n", result.Id, err.Error())
		}
		fmt.Printf("retweeted status with id: %d, created_at %s\n", result.Id, result.CreatedAt.String())
	}
}

func LoadConfig() *ConfData {
	conf := ConfData{}
	buf, err := ioutil.ReadFile(*configFile)
	if err != nil {
		log.Fatal(err)
	}
	if err := yaml.Unmarshal(buf, &conf); err != nil {
		log.Fatal(err)
	}

	return &conf
}

func search(client *http.Client, searchTerm string) ([]byte, error) {
	searchUrl, err := url.Parse("https://api.twitter.com/1.1/search/tweets.json")
	if err != nil {
		return []byte(``), err
	}
	parameters := url.Values{}
	parameters.Set("q", searchTerm)
	searchUrl.RawQuery = parameters.Encode()

	response, err := client.Get(searchUrl.String())
	if err != nil {
		return []byte(``), err
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func retweet(client *http.Client, tweetId int64) error {
	retweetUrl, err := url.Parse("https://api.twitter.com/1.1/statuses/retweet/")
	if err != nil {
		return err
	}

	retweetUrl.Path += fmt.Sprintf("%d.json", tweetId)
	_, err = client.PostForm(retweetUrl.String(), url.Values{})
	if err != nil {
		return err
	}

	return nil
}
