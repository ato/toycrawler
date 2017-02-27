package main

import (
	"fmt"
	"crypto/sha1"
	"os"
	"log"
	"github.com/google/uuid"
	"bytes"
	"github.com/ato/toycrawler/chrome"
	"time"
	"encoding/base32"
)

type WarcWriter struct {
	file *os.File
}

func NewWarcWriter() (*WarcWriter)  {
	filename := "data/test.warc"
	file, err := os.OpenFile(filename, os.O_CREATE | os.O_WRONLY, 0644)

	if err != nil {
		log.Fatalf("error opening %s for writing: %s\n", filename, err)
	}

	w := new(WarcWriter)
	w.file = file
	return w
}

func newRecordId() string {
	id, err := uuid.NewRandom()
	if err != nil {
		log.Fatalf("error generating uuid: %s\n", err)
	}
	return id.URN()
}

func formatHeaders(headers map[string]string) string {
	var buffer bytes.Buffer

	for k,v := range headers {
		buffer.WriteString(k)
		buffer.WriteString(": ")
		buffer.WriteString(v)
		buffer.WriteString("\r\n")
	}

	return buffer.String()
}

func (w *WarcWriter) WriteExchange(exchange *chrome.Exchange)  {
	log.Println("writing exchange")
	requestId := newRecordId()
	responseId := newRecordId()
	date := time.Now().In(time.UTC).Format(time.RFC3339)

	requestHeader := exchange.Response.RequestHeadersText
	if requestHeader == "" {
		requestHeader = fmt.Sprintf("%s %s HTTP/1.1\r\n%s",
			exchange.Request.Method, exchange.Request.Url,
			formatHeaders(exchange.Response.RequestHeaders))
	}

	fmt.Fprintf(w.file, "WARC/1.0\r\n" +
		"WARC-Type: request\r\n" +
		"WARC-Target-URI: %s\r\n" +
		"WARC-Date: %s\r\n" +
		"WARC-Concurrent-To: <%s>\r\n" +
		"WARC-Record-ID: <%s>\r\n" +
		"Content-Type: application/http;msgtype=request\r\n" +
		"Content-Length: %d\r\n\r\n%s\r\n",
		exchange.Response.Url, date, requestId, responseId, len(requestHeader), requestHeader)

	responseHeader := exchange.Response.HeadersText
	if responseHeader == "" {
		// XXX: pywb is confused by protocols like SPDY so force HTTP
		protocol := "HTTP/1.1"
		if (exchange.Response.Protocol == "http/1.0") {
			protocol = "HTTP/1.0"
		}
		responseHeader = fmt.Sprintf("%s %d %s\r\n%s",
			protocol, exchange.Response.Status,
			exchange.Response.StatusText, formatHeaders(exchange.Response.Headers))
	}

	responseLength := len(responseHeader) + len(exchange.ResponseBody) + 2

	digestBytes := sha1.Sum(exchange.ResponseBody)
	digest := base32.StdEncoding.EncodeToString(digestBytes[:])

	fmt.Fprintf(w.file, "WARC/1.0\r\n" +
		"WARC-Type: response\r\n" +
		"WARC-Target-URI: %s\r\n" +
		"WARC-Date: %s\r\n" +
		"WARC-Record-ID: <%s>\r\n" +
		"WARC-Payload-Digest: sha1:%s\r\n" +
		"WARC-IP-Address: %s\r\n" +
		"Content-Type: application/http;msgtype=response\r\n" +
		"Content-Length: %d\r\n\r\n%s\r\n",
		exchange.Response.Url, date, responseId, digest, exchange.Response.RemoteIPAddress,
		responseLength, responseHeader)

	w.file.Write(exchange.ResponseBody)
	w.file.WriteString("\r\n")
	w.file.Sync()
}

