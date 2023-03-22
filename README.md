# search-go
Example search application in go

Our goal is to build a sample application in Go that uses Elasticsearch. For our data, we'll use the [Goodreads books dataset](https://sites.google.com/eng.ucsd.edu/ucsdbookgraph/home)[^1].

For reference, I'm using go version 1.20.2. We plan to make two command line applications `load-books` and `search-books` to interact with the dataset and our Elasticsearch cluster.

## Initializing the project

Let's start a  from scratch

```bash
go mod init github.com/nickcanz/search-go

mkdir -p cmd/load-books
touch cmd/load-books/main.go
```

Let's populate `cmd/load-books/main.go` with the following code:

```go
package main

import "fmt"

func main() {
	fmt.Println("Hello from load-books")
}
```

Let's build our project and create the `load-books` executeable that we can run.

```bash
go build ./cmd/load-books
./load-books
Hello from load-books
```

## Connecting to Elasticsearch

We're now ready to connect to our Elasticsearch cluster. Let's put the necessary info from the `Credentials` page into environment variables via a `.env` file.

```bash
ES_URL=<base url>
ES_USER=<access key>
ES_PASSWORD=<access secret>
```

We can keep our cluster specific information in this `.env` to keep it separate from the code. Now we can connect to our Elasticsearch cluster.

```go
package main

import (
	"fmt"
	"log"
	"os"

	elasticsearch7 "github.com/elastic/go-elasticsearch/v7"
	"github.com/joho/godotenv"
)

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

	response, err := client.Info()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Cluster response: %s", response)
}
```

We can pull in our dependencies by running `go mod tidy`. In order to connect to our `v7.10` Elasticsearch cluster, we also need to use the `7.10` version of `go-elasticsearch`. Let's modify our `go.mod` file to reflect this

```go
module github.com/nickcanz/search-go

go 1.20

require (
	github.com/elastic/go-elasticsearch/v7 v7.10.0
	github.com/joho/godotenv v1.5.1
)
```

And run `go mod tidy` again.

Now we can build our application and see the output:

```bash
go build ./cmd/load-books
./load-books
Hello from load-books
2023/03/21 22:32:17 Cluster response: [200 OK] {
  "name" : "node-name",
  "cluster_name" : "elasticsearch",
  "cluster_uuid" : "<uuid>",
  "version" : {
    "number" : "7.10.2",
    "build_flavor" : "oss",
    "build_type" : "tar",
    "build_hash" : "747e1cc71def077253878a59143c1f785afa92b9",
    "build_date" : "2021-01-13T00:42:12.435326Z",
    "build_snapshot" : false,
    "lucene_version" : "8.7.0",
    "minimum_wire_compatibility_version" : "6.8.0",
    "minimum_index_compatibility_version" : "6.0.0-beta1"
  },
  "tagline" : "You Know, for Search"
}
```

## Loading the dataset

Let's start to write some data! We download the [goodreads_books.json.gz](https://sites.google.com/eng.ucsd.edu/ucsdbookgraph/home) to the root of our project. Let's `gunzip goodreads_books.json.gz` to uncompress the file. The file consists of JSON blobs containing information about a book separated by a new line. The full file is 8.6 gigabytes! Let's cut that down a bit by using the first 1,000 lines

```bash
head -n1000 goodreads_books.json > goodreads_books.1000.json
```

Let's model our data using the following struct:

```go
type Book struct {
	Title       string `json:"title"`
	Url         string `json:"url"`
	Description string `json:"description"`
}
```

This will match up to the index we make with the following settings and mappings:

```
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
```

We can build this up by reading the file and using the [BulkIndexer](https://pkg.go.dev/github.com/elastic/go-elasticsearch/v7@v7.10.0/esutil#BulkIndexer) available in the `esutil` package of `go-elasticsearch`.

```go
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

	file, err := os.Open("goodreads_books.1000.json")
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
```

After building the project and running it again, we can verify all the documents are there. Looking at the Console page of our cluster, we can run `GET /_cat/indices?v`

```
health status index                   uuid pri rep docs.count docs.deleted store.size pri.store.size
green  open   books hB1w4JFFRYuFIxsFhzlV5Q   1   1       1000            0      2.1mb            1mb
```

And we see 1,000 documents in the index!




## References

[^1]:
* Mengting Wan, Julian McAuley, ["Item Recommendation on Monotonic Behavior Chains"](https://github.com/MengtingWan/mengtingwan.github.io/raw/master/paper/recsys18_mwan.pdf), in RecSys'18.  [[bibtex]](https://dblp.uni-trier.de/rec/conf/recsys/WanM18.html?view=bibtex)
* Mengting Wan, Rishabh Misra, Ndapa Nakashole, Julian McAuley, ["Fine-Grained Spoiler Detection from Large-Scale Review Corpora"](https://aclanthology.org/P19-1248), in ACL'19. [[bibtex]](https://dblp.uni-trier.de/rec/conf/acl/WanMNM19.html?view=bibtex)
