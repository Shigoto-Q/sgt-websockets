package docker

import (
	"context"
    "time"
	"fmt"
    "io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)


var dockerRegistryID = "shigoto"

func GetDockerClient() *client.Client {
  cli, err := client.NewClientWithOpts(client.WithHost("tcp://shigoto_dind:2375", client.WithAPIVersionNegotiation()))
  if err != nil {
    log.Println(err)
  }
  return &cli
}


func BuildImage(client *client.Client, imageName string, fileContext *io.Reader) error {
  ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
  defer cancel()

  imageOptions := types.ImageBuildOptions{
    Dockerfile: "Dockerfile",
    Tags: []string{dockerRegistryID + "/" + imageName},
    Remove: true,
  }

  res, err := client.ImageBuild(ctx, fileContext, imageOptions)
  if err != nil {
    return err
  }
  defer res.Body.close()
  return nil
}


func PushImage(client *client.Client, authConfig *types.AuthConfig, imageName string) error {
  ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
  defer cancel()

  authConfigBytes, err := json.Marshall(authConfig)
  if err != nil {
    return nil
  }
  authConfigEncoded := base64.URLEncoding.EncodeToString(authConfigBytes)

  // Perhaps add random has at the end
  tag := dockerRegistryID + "/" + imageName
  pushOptions := types.ImagePushOptions{
    RegistryAuth: authConfigEncoded,
  }
  res, err := client.ImagePush(ctx, tag, pushOptions)
  if err != nil {
    return nil
  }
  defer res.close()
  return nil
}
