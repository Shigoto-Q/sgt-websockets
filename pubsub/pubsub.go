package pubsub

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/Shigoto-Q/sgt-websockets/docker"
	"github.com/Shigoto-Q/sgt-websockets/git"
	"github.com/Shigoto-Q/sgt-websockets/utils"

	"github.com/gomodule/redigo/redis"
	"github.com/gorilla/websocket"
)

const (
	PUBLISH      = "publish"
	SUBSCRIBE    = "subscribe"
	UNSUBSCRIBE  = "unsubscribe"
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

type EndMessage struct {
	Status string `json:"status"`
}

type Message struct {
	Action string          `json:"action"`
	Topic  string          `json:"topic"`
	Data   json.RawMessage `json:"data"`
	Token  string          `json:"token"`
}

type Image struct {
	Repository string `json:"Repository"`
	Name       string `json:"Name"`
	ImageName  string `json:"ImageName"`
	Command    string `json:"Command"`
}

var dockerRegistryID = "shigoto"

type Subscription struct {
	Topic  string
	Client *Client
	UserId string
	sync.Mutex
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
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

func buildImage(client *client.Client, imageName string, fileContext io.Reader, wsClient *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	imageOptions := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{dockerRegistryID + "/" + imageName},
		Remove:     true,
	}

	res, err := client.ImageBuild(ctx, fileContext, imageOptions)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	err = send(res.Body, wsClient)
	if err != nil {
		return err
	}
	return nil
}

func send(rd io.Reader, client *Client) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {
		lastLine = scanner.Text()
		client.Send([]byte(scanner.Text()))
	}

	errLine := &ErrorLine{}
	json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func handleCreateImage(client *Client, image *Image, m *Message) {
	dockerClient := docker.GetDockerClient()
	auth := docker.LoadCredentials()
	resp := CreateTask(image, m.Token)
	client.Send(resp)
	path := git.CloneRepo(image.Repository, image.ImageName)
	fileContext := git.GetContext(path)
	err := buildImage(dockerClient, image.ImageName, fileContext, client)
	if err != nil {
		log.Fatal(err)
	}
	err = os.RemoveAll(path)
	if err != nil {
		log.Fatal(err)
	}
	err = pushImage(dockerClient, auth, image.ImageName, client)
	if err != nil {
		log.Fatal(err)
	}
	endMsg := EndMessage{
		Status: "This is the end",
	}
	msg, err := json.Marshal(endMsg)
	if err != nil {
		log.Fatal(err)
	}
	client.Send(msg)
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
		fmt.Println("Client wants to create new docker image", m.Topic, client.Id)
		if err != nil {
			log.Fatal(err)
		}
		handleCreateImage(&client, &image, &m)
	default:
		break
	}

	return ps
}

func pushImage(client *client.Client, authConfigEncoded string, imageName string, wsClient *Client) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()
	// Perhaps add random has at the end
	tag := dockerRegistryID + "/" + imageName
	pushOptions := types.ImagePushOptions{
		RegistryAuth: authConfigEncoded,
	}
	res, err := client.ImagePush(ctx, tag, pushOptions)
	defer res.Close()
	err = send(res, wsClient)
	if err != nil {
		return err
	}
	if err != nil {
		return nil
	}
	return nil
}
