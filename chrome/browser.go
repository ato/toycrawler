package chrome

import (
	"log"
	"encoding/json"
	"time"
	"encoding/base64"
)

type Browser struct {
	client *Client
	ExchangeWriter func(exchange *Exchange)
}

type Exchange struct {
	Request *NetworkRequest
	Response *NetworkResponse
	ResponseBody []byte
}

const jsExtractLinks = `
	(function() {
		var anchors = document.querySelectorAll('a');
		var links = [];
		for (var i = 0; i < anchors.length; i++) {
			links.push(anchors[i].href);
		}

		// XXX: old versions of prototype.js override toJSON() breaking JSON.stringify
		var arrayToJSON = Array.prototype.toJSON;
		var stringToJSON = String.prototype.toJSON;
		delete Array.prototype.toJSON;
		delete String.prototype.toJSON;

		var json = JSON.stringify(links);

		Array.prototype.toJSON = arrayToJSON;
		String.prototype.toJSON = stringToJSON;

		return json;
	})()
`

func ExtractLinks(c *Client) ([]string, error) {
	var response struct {
		Result struct{
			Value string
		}
	}
	err := c.Call("Runtime.evaluate", map[string]interface{}{"expression": jsExtractLinks}, &response)
	if (err != nil) {
		return nil, err
	}

	var links []string
	if err = json.Unmarshal([]byte(response.Result.Value), &links); err != nil {
		return nil, err
	}

	return links, nil
}

func Connect(host string, port int32) (*Browser, error) {
	c, err := DialFirstTab(host, port)
	if err != nil {
		return nil, err
	}

	c.Call("Page.enable", nil, nil)
	c.Call("Network.enable", nil, nil)

	c.Call("Network.addBlockedURL", map[string]interface{}{"url": "http://www.google-analytics.com/ga.js"}, nil)
	c.Call("Network.addBlockedURL", map[string]interface{}{"url": "https://ssl.google-analytics.com/ga.js"}, nil)

	return &Browser{client: c}, nil
}

func getResponseBody(c *Client, requestId string) []byte {
	var result struct{Body string; Base64Encoded bool}

	err := c.Call("Network.getResponseBody",
		map[string]interface{}{"requestId": requestId}, &result)

	if err != nil {
		log.Fatalf("error getting response body: %s\n", err)
	}

	if result.Base64Encoded {
		body, err := base64.StdEncoding.DecodeString(result.Body)
		if err != nil {
			log.Fatalf("error base64 decoding response body: %s\n", err)
		}
		return body
	} else {
		return []byte(result.Body)
	}
}

func (b *Browser) Browse(url string) (links []string) {
	c := b.client
	timeout := time.After(time.Second * 10)
	c.Events = make(chan interface{}, 1000)

	if err := c.Call("Page.navigate", map[string]interface{}{"url": url}, nil); err != nil {
		log.Fatal(err)
	}

	defer b.client.Call("Page.navigate", map[string]interface{}{"url": "about:blank"}, nil)

	var firstRequestId string
	inflightRequests := map[string]*NetworkRequest{}
	inflightResponses := map[string]*NetworkResponse{}

	for {
		select {
		case event := <-c.Events:
			switch event.(type) {
			case *PageLoadEventFired:
				var err error
				links, err = ExtractLinks(c)
				if (err != nil) {
					log.Fatalf("link extraction failed: %s\n", err)
				}
				return links

			case *NetworkRequestWillBeSent:
				e := event.(*NetworkRequestWillBeSent)
				if firstRequestId == "" {
					firstRequestId = e.RequestId
				}
				inflightRequests[e.RequestId] = e.Request

			case *NetworkResponseReceived:
				e := event.(*NetworkResponseReceived)
				inflightResponses[e.RequestId] = e.Response

			case *NetworkLoadingFailed:
				if event.(*NetworkLoadingFailed).RequestId == firstRequestId {
					log.Printf("loading first request failed\n")
					return nil
				}

			case *NetworkLoadingFinished:
				if b.ExchangeWriter != nil {
					requestId := event.(*NetworkLoadingFinished).RequestId
					if response, ok := inflightResponses[requestId]; ok {
						exchange := new(Exchange)
						exchange.Request = inflightRequests[requestId]
						exchange.Response = response
						exchange.ResponseBody = getResponseBody(c, requestId)
						b.ExchangeWriter(exchange)
					}
				}
			}
		case <-timeout:
			log.Fatal("Timed out")
			return nil
		}
	}
}

func (b *Browser) Close() {
	b.client.Close()
}