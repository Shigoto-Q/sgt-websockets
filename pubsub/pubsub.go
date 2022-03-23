package pubsub

import (
	"encoding/json"
    "bytes"
    "net/http"
	"fmt"
    "io"
	"log"
	"sync"

	"github.com/Shigoto-Q/sgt-websockets/git"
	"github.com/Shigoto-Q/sgt-websockets/utils"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

const (
	PUBLISH     = "publish"
	SUBSCRIBE   = "subscribe"
	UNSUBSCRIBE = "unsubscribe"
    CREATE_IMAGE = "create-image"
)

type PubSub struct {
	Clients       []Client
	Subscriptions []Subscription
}

type Client struct {
	Id         string
	Connection *websocket.Conn
}


type TestMessage struct {
  Name string `json:"name"`
}


type Message struct {
	Action  string          `json:"action"`
	Topic   string          `json:"topic"`
	Data json.RawMessage `json:"data"`
	Token   string          `json:"token"`
}


type Image struct {
  Repository string `json:"Repository"`
  Name string `json:"Name"`
  ImageName string `json:"ImageName"`
  Command string `json:"Command"`
}


type Subscription struct {
	Topic  string
	Client *Client
	UserId string
	sync.Mutex
}

func (ps *PubSub) AddClient(client Client) *PubSub {
	ps.Clients = append(ps.Clients, client)
	return ps
}

func (ps *PubSub) RemoveClient(client Client) *PubSub {
	for index, sub := range ps.Subscriptions {
		if client.Id == sub.Client.Id {
			ps.Subscriptions = append(ps.Subscriptions[:index], ps.Subscriptions[index+1:]...)
		}
	}
	for index, c := range ps.Clients {
		if c.Id == client.Id {
			ps.Clients = append(ps.Clients[:index], ps.Clients[index+1:]...)
		}
	}
	return ps
}

func (ps *PubSub) GetSubscriptions(topic string, client *Client) []Subscription {
	var subscriptionList []Subscription

	for _, subscription := range ps.Subscriptions {
		if client != nil {
			if subscription.Client.Id == client.Id && subscription.Topic == topic {
				subscriptionList = append(subscriptionList, subscription)
			}
		} else {
			if subscription.Topic == topic {
				subscriptionList = append(subscriptionList, subscription)
			}
		}
	}
	return subscriptionList
}

func (ps *PubSub) Subscribe(client *Client, topic string, gPubSubConn *redis.PubSubConn, token string) *PubSub {
	clientSubs := ps.GetSubscriptions(topic, client)
	if len(clientSubs) > 0 {
		return ps
	}
	userId := utils.GetUser(token)
	newSubscription := Subscription{
		Topic:  topic,
		Client: client,
		UserId: userId,
	}
	if err := gPubSubConn.Subscribe(topic); err != nil {
		log.Panic(err)
	}
	newSubscription.Lock()
	ps.Subscriptions = append(ps.Subscriptions, newSubscription)
	defer newSubscription.Unlock()
	return ps
}

func (ps *PubSub) Publish(topic string, message []byte, excludeClient *Client) {
	subscriptions := ps.GetSubscriptions(topic, nil)
	for _, sub := range subscriptions {
		err := sub.Client.Send(message)
		if err != nil {
			log.Panic(err)
		}
	}
}

func (client *Client) Send(message []byte) error {
	return client.Connection.WriteMessage(1, message)

}

func (ps *PubSub) Unsubscribe(client *Client, topic string) *PubSub {
	//clientSubscriptions := ps.GetSubscriptions(topic, client)
	for index, sub := range ps.Subscriptions {
		if sub.Client.Id == client.Id && sub.Topic == topic {
			// found this subscription from client and we do need remove it
			ps.Subscriptions = append(ps.Subscriptions[:index], ps.Subscriptions[index+1:]...)
		}
	}

	return ps

}

var url = "http://django:8000/api/v1/docker/images/create/"

func CreateTask(image *Image, token string) []byte {
  client := &http.Client{}
  payloadBuffer := new(bytes.Buffer)
  json.NewEncoder(payloadBuffer).Encode(image)
  req, err := http.NewRequest("POST", url, payloadBuffer)
  if err != nil {
    log.Fatal(err)
  }
  req.Header.Add("Authorization", token)
  resp, err := client.Do(req)
  if err != nil {
    log.Fatal(err)
  }
  bodyBytes, err := io.ReadAll(resp.Body)
  if err != nil {
    log.Fatal(err)
  }
  return bodyBytes
}


func handleCreateImage(client *Client, image *Image, m *Message) {
  resp := CreateTask(image, m.Token)
  client.Send(resp)
  path := git.CloneRepo(image.Repository, image.ImageName)
  ctx := git.GetContext(path)
  log.Println(ctx)
}

func (ps *PubSub) HandleReceiveMessage(client Client, messageType int, payload []byte, gPubSubConn *redis.PubSubConn) *PubSub {
	m := Message{}
	err := json.Unmarshal(payload, &m)
	if err != nil {
		fmt.Println("This is not correct message payload")
		return ps
	}
	switch m.Action {
	case PUBLISH:
		ps.Publish(m.Topic, m.Data, nil)
	case SUBSCRIBE:
		ps.Subscribe(&client, m.Topic, gPubSubConn, m.Token)
	case UNSUBSCRIBE:
		fmt.Println("Client want to unsubscribe the topic", m.Topic, client.Id)
    case CREATE_IMAGE:
      var image Image
      err = json.Unmarshal(m.Data, &image)
	  fmt.Println("Client wants to create new docker image", m.Topic, client.Id, image)
      if err != nil {
        panic(err)
      }

      // TODO Handle creating and pushing images
      handleCreateImage(&client, &image, &m)
	default:
		break
	}

	return ps
}
