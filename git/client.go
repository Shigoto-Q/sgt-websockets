package git

import (
  "os"
  "log"
  "io"

  "github.com/go-git/go-git/v5"
  "github.com/mitchellh/go-homedir"
  "github.com/docker/docker/pkg/archive"
)


var basePath = "/tmp/repositories/"

func createFullPath(name string) string {
  return basePath + name + "/"
}

func CloneRepo(url string, name string) string {
  // TODO Add user path
  path := createFullPath(name)
  _, err := git.PlainClone(path, false, &git.CloneOptions{
    URL: url,
    Progress: os.Stdout,
  })
  if err != nil {
    log.Fatal(err)
  }
  return path
}

func GetContext(path string) io.Reader {
	filePath, err := homedir.Expand(path)
	if err != nil {
      log.Println(err)
	}
	ctx, _ := archive.TarWithOptions(filePath, &archive.TarOptions{})
	return ctx
}
