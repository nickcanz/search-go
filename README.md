# search-go
Example search application in go

Our goal is to build a sample application in Go that uses Elasticsearch. For our data, we'll use the [Goodreads books dataset](https://sites.google.com/eng.ucsd.edu/ucsdbookgraph/home)[^1].

For reference, I'm using go version 1.20.2. We plan to make two command line applications `load-books` and `search-books` to interact with the dataset and our Elasticsearch cluster.

## Initializing the project

Let's start a  from scratch

```
go mod init github.com/nickcanz/search-go

mkdir -p cmd/load-books
touch cmd/load-books/main.go
```

Let's populate `cmd/load-books/main.go` with the following code:

```
package main

import "fmt"

func main() {
	fmt.Println("Hello from load-books")
}
```

Build our project with `go build ./cmd/load-books` produces the `load-books` executeable that we can run.

```
./load-books
Hello from load-books
```

## Connecting to Elasticsearch

TODO: make .env file, load go module for godotenv and correct version of go-elasticsearch


## Loading the dataset

We download the [goodreads_books.json.gz](https://sites.google.com/eng.ucsd.edu/ucsdbookgraph/home) to the root of our project.



## References

[^1]:
* Mengting Wan, Julian McAuley, ["Item Recommendation on Monotonic Behavior Chains"](https://github.com/MengtingWan/mengtingwan.github.io/raw/master/paper/recsys18_mwan.pdf), in RecSys'18.  [[bibtex]](https://dblp.uni-trier.de/rec/conf/recsys/WanM18.html?view=bibtex)
* Mengting Wan, Rishabh Misra, Ndapa Nakashole, Julian McAuley, ["Fine-Grained Spoiler Detection from Large-Scale Review Corpora"](https://aclanthology.org/P19-1248), in ACL'19. [[bibtex]](https://dblp.uni-trier.de/rec/conf/acl/WanMNM19.html?view=bibtex)
