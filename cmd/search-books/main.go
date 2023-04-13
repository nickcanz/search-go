package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/joho/godotenv"
)

type Book struct {
	Title       string `json:"title"`
	Url         string `json:"url"`
	Description string `json:"description"`
}

type BookSearchResponse struct {
	Took float64 `json:"took"`
	Hits struct {
		Hits []struct {
			Book  Book    `json:"_source"`
			Score float64 `json:"_score"`
		} `json:"hits"`
	} `json:"hits"`
}

func main() {
	queryPtr := flag.String("query", "", "Query to search for")
	flag.Parse()
	if *queryPtr == "" {
		log.Fatalf("No query provided for -query parameter")
	}

	fmt.Printf("Searching books for: %s\n", *queryPtr)

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cfg := elasticsearch7.Config{
		Addresses: []string{
			os.Getenv("ES_URL"),
		},
		Username: os.Getenv("ES_USER"),
		Password: os.Getenv("ES_PASSWORD"),
	}

	client, err := elasticsearch7.NewClient(cfg)
	if err != nil {
		log.Fatal(err)
	}
	query := fmt.Sprintf(` {
	   "query": {
	   	"multi_match":{
		  "query":"%s",
		  "fields": [ "title", "url", "description" ]
		}
	   },
	   "size": 10
	}`, *queryPtr)
	resp, err := client.Search(
		client.Search.WithIndex("books"),
		client.Search.WithBody(strings.NewReader(query)))
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		log.Fatalf("Error querying, status: %s, response body: %s", resp.Status(), resp.String())
	}

	var bookSearchResponse BookSearchResponse

	err = json.NewDecoder(resp.Body).Decode(&bookSearchResponse)
	if err != nil {
		log.Fatal(err)
	}

	for _, bookHit := range bookSearchResponse.Hits.Hits {
		fmt.Printf("%s, %s with score of %f\n", bookHit.Book.Title, bookHit.Book.Url, bookHit.Score)
	}
}
