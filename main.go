package main

import (
	"awesomeProject1/pubsub"
	"awesomeProject1/types"
	"encoding/json"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
	"strconv"
	"time"
)

var redisPool *redis.Pool
var taskResult types.TaskResult
var taskCount types.TaskCountMessage

var (
	gPubSubConn *redis.PubSubConn
	gRedisConn  = func() (redis.Pool, error) {
		return redis.Pool{
			MaxIdle:     3,
			IdleTimeout: 240 * time.Second,
			Dial: func() (redis.Conn, error) {
				c, err := redis.Dial("tcp", "redis:6379")
				if err != nil {
					return nil, err
				}
				return c, err
			},
		}, nil
	}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func autoId() string {
	id := uuid.NewV4().String()
	return id
}

var ps = &pubsub.PubSub{}

func websocketHandler(w http.ResponseWriter, r *http.Request) {
	upgrader.CheckOrigin = func(r *http.Request) bool {
		return true

	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := pubsub.Client{
		Id:         autoId(),
		Connection: conn,
	}

	// add this client into the list
	ps.AddClient(client)
	log.Println("New Client is connected, total: ", len(ps.Clients))

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("Something went wrong", err)
			ps.RemoveClient(client)
			log.Println("total clients and subscriptions ", len(ps.Clients), len(ps.Subscriptions))
			return
		}
		go listenToMessages()
		go ps.HandleReceiveMessage(client, messageType, p, gPubSubConn)
	}
}

func sendTaskResultMessage(data []byte, sub pubsub.Subscription) {
	err := json.Unmarshal([]byte(data), &taskResult)
	if err != nil {
		log.Panic(err)
	}
	if sub.UserId == strconv.FormatInt(int64(taskResult.User_id), 10) {
		err = sub.Client.Send([]byte(data))
		if err != nil {
			log.Println(err)
		}
	}
}

func sendTaskCountMessage(data []byte, sub pubsub.Subscription) {
	err := json.Unmarshal(data, &taskCount)
	if err != nil {
		log.Panic(err)
	}
	if sub.UserId == strconv.FormatInt(int64(taskCount.UserId), 10) {
		err = sub.Client.Send(data)
		if err != nil {
			log.Println(err)
		}
	}
}

func listenToMessages() {
	for {
		switch v := gPubSubConn.Receive().(type) {
		case redis.Message:
			log.Printf("Received message from %s", v.Channel)
			subscriptions := ps.GetSubscriptions(v.Channel, nil)
			for _, sub := range subscriptions {
				if v.Channel == types.TaskResults {
					go sendTaskResultMessage(v.Data, sub)
				} else if v.Channel == types.TaskCount {
					go sendTaskCountMessage(v.Data, sub)
				}
			}
		case redis.Subscription:
			log.Printf("Subscription message: %s: %s %d\n", v.Channel, v.Kind, v.Count)
		case error:
			log.Println("Error pub/sub, delivery stopped")
			return
		}
	}
}

func main() {
	gRedisConn, err := gRedisConn()
	if err != nil {
		log.Panic(err)
	}
	gPubSubConn = &redis.PubSubConn{Conn: gRedisConn.Get()}
	log.Println("Server is running: http://localhost:8080")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		http.ServeFile(w, r, "static")

	})
	http.HandleFunc("/ws", websocketHandler)
	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}
}
