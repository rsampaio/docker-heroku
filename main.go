package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/dotcloud/docker"
	dockerClient "github.com/rsampaio/go-dockerclient"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"strconv"
)

var fUser = flag.String("username", "", "heroku username")
var fToken = flag.String("token", "", "heroku account token")
var fApp = flag.String("app", "", "heroku app name")
var fRelease = flag.Int("release", 1, "application release")
var fProcess = flag.String("process", "web", "process type")

var baseUrl = "https://api.heroku.com"
var appName string

type Releases struct {
	Id   string
	Slug map[string]interface{}
}

type Volume struct {
	string
}

type Slug struct {
	Blob          map[string]interface{}
	Process_Types map[string]interface{}
}

func HttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{},
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}
	return client
}

func MakeRequest(appName string, what string, id string) *http.Request {
	requestUrl := fmt.Sprintf("%s/apps/%s/%s/%s", baseUrl, appName, what, id)
	request, err := http.NewRequest("GET", requestUrl, nil)
	if err != nil {
		panic("request create")
	}
	request.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	request.SetBasicAuth(*fUser, *fToken)
	return request
}

func GetSlugIdForRelease(appName string, release_name string) string {
	client := HttpClient()
	request := MakeRequest(appName, "releases", release_name)
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var release Releases
	err = json.Unmarshal(body, &release)
	if err != nil {
		panic(err)
	}
	return release.Slug["id"].(string)
}

func GetSlugBlobUrl(appName string, slugId string) Slug {
	client := HttpClient()
	request := MakeRequest(appName, "slugs", slugId)
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	var slug Slug
	err = json.Unmarshal(body, &slug)
	if err != nil {
		panic(err)
	}
	return slug
}

func FetchSlugArchive(appName string, slug_url string) *bytes.Buffer {
	buf := new(bytes.Buffer)
	client := HttpClient()
	request, err := http.NewRequest("GET", slug_url, nil)
	resp, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	buf.Write(body)
	return buf
}

func UntarFiles(buffer *bytes.Buffer) {
	os.RemoveAll("app")
	r, _ := gzip.NewReader(buffer)
	tr := tar.NewReader(r)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}

		if err != nil {
			log.Fatalln(err)
		}

		os.MkdirAll(path.Dir(hdr.Name), os.FileMode(0775))
		if hdr.Linkname != "" {
			err = os.Symlink(hdr.Linkname, hdr.Name)
			if err != nil {
				panic(err)
			}
		} else {
			fileContent, _ := ioutil.ReadAll(tr)
			ioutil.WriteFile(hdr.Name, fileContent, os.FileMode(0775))
		}
	}
}

func RunDockerContainer(cmd string) {
	herokuClient := HttpClient()
	request := MakeRequest(appName, "config-vars", "")
	resp, err := herokuClient.Do(request)
	if err != nil {
		panic("do request")
	}

	herokuEnv, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic("read response")
	}
	fmt.Println("Heroku Env:", string(herokuEnv))
	
	cwd, _ := os.Getwd()

	dClient, err := dockerClient.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		panic("new client")
	}

	var config docker.Config
	exposedPorts := make(map[docker.Port]struct{})
	exposedPorts["80"] = struct{}{}

	volumes := make(map[string]struct{})
	volumes["/app"] = struct{}{}

	config.Volumes = volumes
	config.ExposedPorts = exposedPorts
	config.Cmd = []string{"/bin/bash", "-c", cmd}

	config.Env = []string{"PORT=80"}

	config.Image = "heroku-runtime"
	config.WorkingDir = "/app"

	container, err := dClient.CreateContainer(dockerClient.CreateContainerOptions{}, &config)
	if err != nil {
		panic("create container")
	}

	err = dClient.StartContainer(container.ID, &docker.HostConfig{Binds: []string{cwd + "/app:/app"}})
	if err != nil {
		panic("start container")
	}
	fmt.Fprintf(os.Stdout, "%s\n", container.ID)
}

func main() {
	flag.Parse()

	if flag.NFlag() < 5 {
		flag.PrintDefaults()
		os.Exit(-1)
	}
	appName := *fApp
	release_name := strconv.Itoa(*fRelease)
	slugId := GetSlugIdForRelease(appName, release_name)
	fmt.Fprintf(os.Stderr, "Slug ID: %s\n", slugId)

	slug := GetSlugBlobUrl(appName, slugId)
	fmt.Fprintf(os.Stderr, "Slug URL: %s\n", slug.Blob["get"].(string))

	tarBuffer := FetchSlugArchive(appName, slug.Blob["get"].(string))
	fmt.Fprintf(os.Stderr, "Download complete, unpacking.\n")

	UntarFiles(tarBuffer)
	fmt.Fprintf(os.Stderr, "Unpack complete\n")
	RunDockerContainer(slug.Process_Types[*fProcess].(string))

}