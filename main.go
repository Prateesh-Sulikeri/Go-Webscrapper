package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

type Result struct {
	URL string
	Title string
	Error error 
}

func generateURLs(urls []string) <- chan string {
	jobs := make(chan string)
	
	go func ()  {
		for _, url := range urls {
			jobs <- url
		}
		close(jobs)	
	}()
	return jobs
}

func worker(ctx context.Context, id int, jobs <- chan string, results chan <- Result, wg *sync.WaitGroup) {
	defer wg.Done()


	for {
		select {
		case <-ctx.Done():
			return
		case url, ok := <-jobs:
			if !ok {
				return
			}
			r := Result {URL: url}
			resp, err := http.Get(url)
			if err != nil {
				r.Error = err
				results <- r
				fmt.Println("HTTP Get Failed: ", err)
				continue
			}

			if resp.StatusCode != http.StatusOK {
				r.Error = fmt.Errorf("status: %d", resp.StatusCode)
				results <- r
				fmt.Println("Received non OK status: ", resp.StatusCode)
				continue
			}

			title, err := extractTitle(resp.Body)
			resp.Body.Close()
			if err != nil {
				r.Error = err
			} else {
				r.Title = title
			}
			results <- r
		}

	}
}


func extractTitle(body io.Reader) (string, error) {
	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}
	html := string(data)

	start := strings.Index(html, "<title>")
	if start == -1 {
		return  "", fmt.Errorf("no <title> tag found ")
	}

	start += len("<title>")

	end := strings.Index(html[start:], "</title>")
	if end == -1 {
		return "", fmt.Errorf("no </title> tag found")
	}
	return html[start: start+end], nil
}

func main() {

	urls := []string{
        "https://example.com",
        "https://golang.org",
        "https://google.com",
        "https://github.com",
        "https://news.ycombinator.com",
    }

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	jobs := generateURLs(urls)

	results := make(chan Result)

	var wg sync.WaitGroup

	for i:=1 ; i<=3 ; i++ {
		wg.Add(1)
		go worker(ctx, i, jobs, results, &wg)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	for r := range results {
		fmt.Printf("\nFor the URL: %v \nTitle is: %v\n", r.URL, r.Title)
	}
}