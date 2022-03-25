package docker

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"

	"github.com/joho/godotenv"
)

var dockerRegistryID = "shigoto"

func LoadCredentials() string {
	err := godotenv.Load(".env")
	if err != nil {
		fmt.Println(err)
	}
	user := os.Getenv("DOCKER_HUB_USER")
	token := os.Getenv("DOCKER_HUB_TOKEN")
	var authConfig = types.AuthConfig{
		Username:      user,
		Password:      token,
		ServerAddress: "https://index.docker.io/v1/",
	}
	authConfigBytes, err := json.Marshal(authConfig)
	if err != nil {
		log.Fatal(err)
	}
	authConfigEncoded := base64.URLEncoding.EncodeToString(authConfigBytes)
	return authConfigEncoded
}

func GetDockerClient() *client.Client {

	cli, err := client.NewClientWithOpts(client.WithHost("tcp://shigoto_dind:2375"), client.WithAPIVersionNegotiation())
	if err != nil {
		log.Println(err)
	}
	return cli
}

func PushImage(client *client.Client, authConfigEncoded string, imageName string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
	defer cancel()

	// Perhaps add random has at the end
	tag := dockerRegistryID + "/" + imageName
	pushOptions := types.ImagePushOptions{
		RegistryAuth: authConfigEncoded,
	}
	res, err := client.ImagePush(ctx, tag, pushOptions)
	log.Println(res)
	if err != nil {
		return nil
	}
	//defer res.close()
	return nil
}
