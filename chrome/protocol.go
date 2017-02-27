// RPC client for Chrome Debugging Protocol

package chrome

import (
	"github.com/gorilla/websocket"
	"log"
	"sync"
	"encoding/json"
	"os"
	"time"
)

type Client struct {
	Log    *log.Logger
	Events chan interface{}
	conn   *websocket.Conn

	mutex      sync.Mutex
	seq        uint64
	pending    map[uint64]*Call
}

type Call struct {
	Id     uint64
	Method string
	Done   chan *Call
	Result *json.RawMessage
	Error  *Error
}

type request struct {
	Id     uint64                 `json:"id"`
	Method string                 `json:"method,omitempty"`
	Params map[string]interface{} `json:"params,omitempty"`
}

type response struct {
	Id     uint64
	Result *json.RawMessage
	Error  *Error
	Method string
	Params json.RawMessage
}

type Error struct {
	Code    int32  `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string {
	return e.Message
}

func (client *Client) handle() {
	for {
		msg := new(response)
		err := client.conn.ReadJSON(msg)
		if err != nil {
			log.Fatal(err)
		}

		if msg.Method != "" { // event
			client.Log.Printf("EVENT  %-26s %s", msg.Method, string(msg.Params))

			client.mutex.Lock()
			constructor, ok := eventConstructors[msg.Method]
			client.mutex.Unlock()

			if ok && client.Events != nil {
				event := constructor()
				err = json.Unmarshal(msg.Params, event)
				if err != nil {
					log.Fatalf("Error unmarshalling event %s: %s\n", msg.Method, err)
				}
				client.Events <- event
			}
		} else { // rpc response
			client.mutex.Lock()
			call := client.pending[msg.Id]
			delete(client.pending, msg.Id)
			client.mutex.Unlock()

			call.Result = msg.Result
			call.Error = msg.Error
			call.Done <- call

			if call.Error == nil {
				client.Log.Printf("RESULT %-26s %s", call.Method, *call.Result)
			} else {
				client.Log.Printf("ERROR  %-26s %s", call.Method, call.Error)
			}
		}
	}
}

func Dial(wsUrl string) (*Client, error) {
	conn, _, err := websocket.DefaultDialer.Dial(wsUrl, nil)
	if err != nil {
		return nil, err
	}

	c := new(Client)
	c.conn = conn
	c.pending = map[uint64]*Call{}
	c.Log = log.New(os.Stdout, "chrome ", log.Lmicroseconds)

	go c.handle()

	return c, nil
}


func (client *Client) Close()  {
	client.conn.Close()
}

// Asynchronously call an RPC method
func (client *Client) Go(method string, params map[string]interface{}, done chan *Call) (*Call, error) {
	client.mutex.Lock()

	id := client.seq
	client.seq++

	req := new(request)
	req.Id = id
	req.Method = method
	req.Params = params

	err := client.conn.WriteJSON(req)
	if err != nil {
		client.mutex.Unlock()
		return nil, err
	}

	call := new(Call)
	call.Id = id
	call.Done = done
	call.Method = method

	client.pending[id] = call

	client.mutex.Unlock()

	client.Log.Printf("CALL   %-26s %s", call.Method, params)

	return call, nil
}

// Synchronously call an RPC method
func (client *Client) Call(method string, params map[string]interface{}, result interface{}) error {
	call, err := client.Go(method, params, make(chan *Call, 1))
	if err != nil {
		return err
	}
	select {
	case <- call.Done:
	case <- time.After(time.Second * 5):
		log.Fatalf("RPC timed out calling %s", method)
	}

	if call.Error != nil {
		return call.Error
	} else if result == nil {
		return nil
	} else {
		return json.Unmarshal(*call.Result, result)
	}
}
