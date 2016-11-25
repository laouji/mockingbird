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

type Mockingbird struct {
	*ConfData
	APIBase string
	APIVer  string
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
	Id        uint64          `json:"id"`
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
	conf := LoadConf()
	mockingbird := NewMockingbird(conf)

	c := oauth.NewConsumer(
		mockingbird.ConsumerKey,
		mockingbird.ConsumerSecret,
		oauth.ServiceProvider{
			RequestTokenUrl:   mockingbird.APIBase + "/oauth/request_token",
			AuthorizeTokenUrl: mockingbird.APIBase + "/oauth/authorize",
			AccessTokenUrl:    mockingbird.APIBase + "/oauth/access_token",
		})

	accessToken := &oauth.AccessToken{
		Token:  mockingbird.AccessToken,
		Secret: mockingbird.TokenSecret,
	}

	client, err := c.MakeHttpClient(accessToken)
	if err != nil {
		log.Fatal(err)
	}

	// search using search_term specified in config file
	rawResults, err := mockingbird.search(client)
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

		err = mockingbird.retweet(client, result.Id)
		if err != nil {
			fmt.Printf("can't retweet status with id: %d, %s\n", result.Id, err.Error())
		}
		fmt.Printf("retweeted status with id: %d, created_at %s\n", result.Id, result.CreatedAt.String())
	}
}

func LoadConf() *ConfData {
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

func NewMockingbird(conf *ConfData) *Mockingbird {
	return &Mockingbird{conf, "https://api.twitter.com", "1.1"}
}

func (m *Mockingbird) search(client *http.Client) ([]byte, error) {
	searchUrl, err := url.Parse(m.APIBase + "/" + m.APIVer + "/search/tweets.json")
	if err != nil {
		return []byte(``), err
	}
	parameters := url.Values{}
	parameters.Set("q", m.SearchTerm)
	searchUrl.RawQuery = parameters.Encode()

	response, err := client.Get(searchUrl.String())
	if err != nil {
		return []byte(``), err
	}
	defer response.Body.Close()

	return ioutil.ReadAll(response.Body)
}

func (m *Mockingbird) retweet(client *http.Client, tweetId uint64) error {
	retweetUrl, err := url.Parse(m.APIBase + "/" + m.APIVer + "/statuses/retweet/")
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
