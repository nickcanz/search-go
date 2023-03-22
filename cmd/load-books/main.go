package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/elastic/go-elasticsearch/v7/esutil"
	"github.com/joho/godotenv"
)

type Book struct {
	Title       string `json:"title"`
	Url         string `json:"url"`
	Description string `json:"description"`
}

func main() {
	fmt.Println("Hello from load-books")

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

	indexName := "books"
	indexBody := `
	{
	  "settings": {
	    "number_of_shards": 1
	  },
	  "mappings": {
	    "properties": {
	      "title": {
	        "type": "text"
	      },
	      "url": {
	        "type": "text"
	      },
	      "description": {
	        "type": "text"
	      }
	    }
	  }
	}`
	_, err = client.Indices.Create(
		indexName,
		client.Indices.Create.WithBody(strings.NewReader(indexBody)),
	)
	if err != nil {
		log.Fatal(err)
	}

	bulkIndexer, err := esutil.NewBulkIndexer(esutil.BulkIndexerConfig{
		Index:      indexName,
		NumWorkers: 1,
		Client:     client,
		ErrorTrace: true,
		OnError: func(ctx context.Context, err error) {
			log.Fatalf("bulkindexer OnError %#v", err)
		},
	})
	if err != nil {
		log.Fatal(err)
	}

	file, err := os.Open("goodreads_books.json")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	reader := bufio.NewReader(file)

	for {
		readBytes, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}

			log.Fatalf("error reading readBytes: %v", err)
			return
		}

		var book Book
		err = json.Unmarshal(readBytes, &book)
		if err != nil {
			log.Fatalf("error unmarshalling json: %v", err)
			return
		}

		documentBytes, err := json.Marshal(book)
		if err != nil {
			log.Fatalf("error marshalling json: %v", err)
			return
		}

		err = bulkIndexer.Add(
			context.Background(),
			esutil.BulkIndexerItem{
				Action: "index",
				Body:   bytes.NewReader(documentBytes),
				// OnFailure is called for each failed operation
				OnFailure: func(ctx context.Context, item esutil.BulkIndexerItem, res esutil.BulkIndexerResponseItem, err error) {
					if err != nil {
						log.Printf("ERROR: %s", err)
					} else {
						log.Printf("ERROR: %s: %s", res.Error.Type, res.Error.Reason)
					}
				},
			})

		if err != nil {
			log.Fatalf("error adding item to bulk indexer: %v", err)
			return
		}
	}
	if err := bulkIndexer.Close(context.Background()); err != nil {
		log.Fatalf("Unexpected error: %s", err)
	}
}
