package main

import (
	"log"
	"github.com/ato/toycrawler/chrome"
	"strings"
)

func main() {
	browser, err := chrome.Connect("localhost", 9292)
	if err != nil {
		log.Fatal(err)
	}
	defer browser.Close()

	warcWriter := NewWarcWriter()
	browser.ExchangeWriter = warcWriter.WriteExchange

	queue := []string{"http://www.nla.gov.au/"}
	seen := map[string]bool{}

	// mark seed urls as seen
	for _, seed := range queue {
		seen[seed] = true
	}

	for len(queue) > 0 {
		target := queue[0]
		queue = queue[1:]

		log.Printf("Browsing %s\n", target)

		links := browser.Browse(target)

		for _, link := range links {
			// enqueue only novel urls
			if !seen[link] && strings.HasPrefix(link, "http://www.nla.gov.au/") {
				queue = append(queue, link)
				seen[link] = true
			}
		}
	}

	browser.Close()
}