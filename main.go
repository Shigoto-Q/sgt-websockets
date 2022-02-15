package main

import (
	"awesomeProject1/pubsub"
	"awesomeProject1/types"
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
	"github.com/satori/go.uuid"
	"log"
	"net/http"
	"strconv"
)

var (
	gPubSubConn *redis.PubSubConn
	gRedisConn  = func() (redis.Conn, error) {
		return redis.Dial("tcp", "redis:6379")
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
	gRedisConn, err := gRedisConn()
	if err != nil {
		log.Panic(err)
	}
	gPubSubConn = &redis.PubSubConn{Conn: gRedisConn}
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

	fmt.Println("New Client is connected, total: ", len(ps.Clients))

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Println("Something went wrong", err)

			ps.RemoveClient(client)
			log.Println("total clients and subscriptions ", len(ps.Clients), len(ps.Subscriptions))

			return
		}
		go listenToMessages()
		ps.HandleReceiveMessage(client, messageType, p, gPubSubConn)
	}
}

var taskResult types.TaskResult
var taskCount types.TaskCountMessage

func sendTaskResultMessage(data []byte, sub pubsub.Subscription) {
	err := json.Unmarshal([]byte(data), &taskResult)
	if err != nil {
		log.Panic(err)
	}
	if sub.UserId == strconv.FormatInt(int64(taskResult.User_id), 10) {
		fmt.Printf("Sending to client id %s message is %s \n", sub.Client.Id, data)
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
		fmt.Printf("Sending to client id %s message is %s \n", sub.Client.Id, data)
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
			subscriptions := ps.GetSubscriptions(v.Channel, nil)
			for _, sub := range subscriptions {
				if v.Channel == types.TaskResults {
					sendTaskResultMessage(v.Data, sub)
				} else if v.Channel == types.TaskCount {
					sendTaskCountMessage(v.Data, sub)
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
	fmt.Println("Server is running: http://localhost:8080")
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {

		http.ServeFile(w, r, "static")

	})

	http.HandleFunc("/ws", websocketHandler)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		return
	}

}
