package main

import "database/sql"
import (
	_ "github.com/go-sql-driver/mysql"
	"log"
	"github.com/ato/toycrawler/chrome"
	"time"
	"net/url"
	"strings"
)

type Database struct {
	conn *sql.DB
}

func OpenDatabase() (*Database) {
	db := new(Database)
	conn, err := sql.Open("mysql", "root:@/toycrawler")
	if err != nil {
		log.Fatalf("Unable to open database: %s\n", err)
	}
	db.conn = conn
	return db
}

func (db *Database) Close() {
	db.conn.Close()
}

func (db *Database) AddLink(srcPageId int64, dstPageUrl string) {
	url, err := url.Parse(dstPageUrl)
	if err != nil {
		log.Printf("Skipping bad link %s: %s\n", dstPageUrl, err)
	}
	protocol := strings.ToLower(url.Scheme)
	host := strings.ToLower(url.Hostname())
	port := url.Port()
	db.conn.Exec("INSERT INTO origin (protocol, host, port) VALUES (?, ?, ?)",
		protocol, host, port)
	db.conn.Exec("INSERT INTO page (origin_id, url) VALUES ((SELECT id FROM origin WHERE protocol = ? AND host = ? AND port = ?), ?)", protocol, host, port, dstPageUrl)
	db.conn.Exec("INSERT INTO link (src_page_id, dst_page_id) VALUES (?, (SELECT id FROM page WHERE url=?))", srcPageId, dstPageUrl)
}

func (db *Database) NextCandidate() (*Page) {
	candidate := new(Page)
	err := db.conn.QueryRow("SELECT id, url FROM page WHERE last_visit IS NULL AND url like 'http://www.canberratimes.com.au/%' ORDER BY ID ASC LIMIT 1").
		Scan(&candidate.id, &candidate.url)
	if err == sql.ErrNoRows {
		return nil
	} else if err != nil {
		log.Fatalf("db.NextCandidate: %s\n", err)
	}
	return candidate
}

func (db *Database) RecordVisit(pageId int64, browsing *chrome.Visit) {
	_, err := db.conn.Exec("UPDATE page SET last_browsed = CURRENT_TIMESTAMP WHERE id = ?", pageId)
	if err != nil {
		log.Fatalf("db.RecordVisit(%d): %s", pageId, err)
	}
	_, err = db.conn.Exec("INSERT INTO browsing (page_id, duration, status, total_bytes, mime_type, dom_text) VALUES (?, ?, ?, ?, ?, ?)",
		pageId, browsing.Duration / time.Millisecond, browsing.Status, browsing.TotalBytes, browsing.MimeType,
		browsing.DomText)
	if err != nil {
		log.Fatalf("db.AddBrowsing2(%d): %s", pageId, err)
	}
}
