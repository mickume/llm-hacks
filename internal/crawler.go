package internal

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

const (
	newLineToken = "\n\n"
)

func SearchAO3(url string, output_file string) (int, error) {
	listCollector := colly.NewCollector()
	listCollector.Limit(&colly.LimitRule{
		DomainGlob:  "*archiveofourown.*",
		Parallelism: 2,
		RandomDelay: 5 * time.Second,
	})
	storyCollector := listCollector.Clone()

	f, err := os.OpenFile(output_file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	storiesFound := 0

	// Find and visit all links
	listCollector.OnHTML("a[href]", func(e *colly.HTMLElement) {
		link := e.Attr("href")
		if strings.Contains(link, "/works/") {
			parts := strings.Split(link, "/")
			if len(parts) == 3 {
				id := parts[2]
				if !strings.ContainsAny(id, "?#") && id != "search" {

					url := fmt.Sprintf("https://archiveofourown.org/works/%s?view_full_work=true&view_adult=true", id)

					err := storyCollector.Visit(url)
					if err != nil {
						fmt.Println(err)
					}
					storiesFound++
				}
			}
		}
	})
	listCollector.OnRequest(func(r *colly.Request) {
		fmt.Printf("Crawling '%s'\n", r.URL)
	})
	listCollector.OnError(func(r *colly.Response, err error) {
		fmt.Println("List Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	// extract the text
	storyCollector.OnHTML("div.userstuff", func(e *colly.HTMLElement) {
		f.WriteString(e.Text)
	})
	storyCollector.OnRequest(func(r *colly.Request) {
		fmt.Printf("Retrieving '%s'\n", r.URL)
	})
	storyCollector.OnError(func(r *colly.Response, err error) {
		fmt.Println("Story Request URL:", r.Request.URL, "failed with response:", r, "\nError:", err)
	})

	err = listCollector.Visit(url)
	return storiesFound, err
}

func RetrieveFromAO3(id, output string) error {
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()

	c := colly.NewCollector()

	c.OnHTML("div.userstuff p", func(e *colly.HTMLElement) {
		f.WriteString(e.Text + newLineToken)
	})

	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Retrieving ", r.URL)
	})

	url := fmt.Sprintf("https://archiveofourown.org/works/%s?view_full_work=true&view_adult=true", id)
	return c.Visit(url)
}
