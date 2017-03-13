package main

import (
	"log"
	"github.com/ato/toycrawler/chrome"
	"strings"
)

type Page struct {
	id int64
	url string
}

func main() {
	db := OpenDatabase()
	defer db.Close()

	browser, err := chrome.Connect("localhost", 9292)
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()

	warcWriter := NewWarcWriter()
	browser.ExchangeWriter = warcWriter.WriteExchange

	for candidate := db.NextCandidate(); candidate != nil; candidate = db.NextCandidate() {

		log.Printf("Visit [%d] %s\n", candidate.id, candidate.url)

		visit := browser.Browse(candidate.url)

		for _, link := range visit.Links {
			if (strings.HasPrefix(link, "http://") ||
				strings.HasPrefix(link, "https://")) {
				db.AddLink(candidate.id, link)
			}
		}

		db.RecordVisit(candidate.id, visit)
	}

	browser.Close()
}

