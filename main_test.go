package main

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	expectedSearchResponse = `{ 'statuses': [{'id':'1111111111111111', 'created_at':'Mon Jan 02 15:04:05 +0000 2006', 'entities':[]}] }`
	dummyConf              = &ConfData{
		SearchTerm:     "#blacklivesmatter",
		ConsumerKey:    "consumerkey",
		ConsumerSecret: "consumersecret",
		AccessToken:    "accesstoken",
		TokenSecret:    "tokensecret",
	}
)

func TestNewMockingbird(t *testing.T) {

	m := NewMockingbird(dummyConf)
	assert.Equal(t, "1.1", m.APIVer, "default API ver is 1.1")
	assert.Equal(t, "https://api.twitter.com", m.APIBase, "default API base url points to twitter")
}

func TestSearch(t *testing.T) {
	handler := func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method, "is a GET request")

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, expectedSearchResponse)
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	m := NewMockingbird(dummyConf)
	m.APIBase = server.URL
	results, err := m.search(&http.Client{})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, expectedSearchResponse, string(results), "response is normal")
}

func TestRetweet(t *testing.T) {
	statusId := uint64(1111111122222)
	handler := func(w http.ResponseWriter, r *http.Request) {
		pathParts := strings.Split(r.URL.Path, "/")
		pathPart := strings.TrimSuffix(pathParts[len(pathParts)-1], ".json")
		fmt.Println(r.URL.Path)
		assert.Equal(t, "POST", r.Method, "is a POST request")
		assert.Equal(t, fmt.Sprint(statusId), pathPart, "id parameter in url is expected id number")

		w.Header().Set("Content-Type", "application/json")
		io.WriteString(w, fmt.Sprintf(`{'id':%d}`, pathPart))
	}

	server := httptest.NewServer(http.HandlerFunc(handler))
	defer server.Close()

	m := NewMockingbird(dummyConf)
	m.APIBase = server.URL
	err := m.retweet(&http.Client{}, statusId)
	if err != nil {
		t.Fatal(err)
	}
}
