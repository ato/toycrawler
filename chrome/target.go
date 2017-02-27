package chrome

import (
	"fmt"
	"encoding/json"
	"net/http"
)

type TargetInfo struct {
	Id                   string
	Title                string
	Type                 string
	Url                  string
	WebSocketDebuggerUrl string
}

func ListTargets(host string, port int32) (targets []TargetInfo, err error)  {
	url := fmt.Sprintf("http://%s:%d/json/list", host, port)

	r, err := http.DefaultClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	json.NewDecoder(r.Body).Decode(&targets)
	return
}

func DialFirstTab(host string, port int32) (*Client, error)  {
	targets, err := ListTargets(host, port)
	if err != nil {
		return nil, err
	}
	return Dial(targets[0].WebSocketDebuggerUrl)
}

func DialNewTab(host string, port int32) (*Client, error)  {
	c, err := DialFirstTab(host, port)
	if err != nil {
		return nil, err
	}

	defer c.Close()

	var result struct{TargetId string}

	err = c.Call("Target.createTarget", map[string]interface{}{"url": "about:blank"}, &result)
	if err != nil {
		return nil, err
	}

	wsUrl := fmt.Sprintf("ws://%s:%d/devtools/pages/%s", host, port, result.TargetId)
	return Dial(wsUrl)
}