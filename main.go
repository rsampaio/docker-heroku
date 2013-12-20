package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"github.com/dotcloud/docker"
	dockerClient "github.com/rsampaio/go-dockerclient"
)

var base_url = "https://api.heroku.com"

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

func MakeRequest(app_name string, what string, id string) *http.Request {
	request_url := fmt.Sprintf("%s/apps/%s/%s/%s", base_url, app_name, what, id)
	request, err := http.NewRequest("GET", request_url, nil)
	if err != nil {
		panic("request create")
	}
	request.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	request.SetBasicAuth(os.Args[1], os.Args[2])
	return request
}

func GetSlugIdForRelease(app_name string, release_name string) string {
	client := HttpClient()
	request := MakeRequest(app_name, "releases", release_name)
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

func GetSlugBlobUrl(app_name string, slug_id string) Slug {
	client := HttpClient()
	request := MakeRequest(app_name, "slugs", slug_id)
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

func FetchSlugArchive(app_name string, slug_url string) *bytes.Buffer {
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
			file_content, _ := ioutil.ReadAll(tr)
			ioutil.WriteFile(hdr.Name, file_content, os.FileMode(0775))
		}
	}
}

func RunDockerContainer(cmd string) {
	cwd, _ := os.Getwd()
	docker_client, err := dockerClient.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		panic("new client")
	}

	var config docker.Config
	exposed_ports := make(map[docker.Port]struct{})
	exposed_ports["80"] = struct{}{}

	volumes := make(map[string]struct{})
	volumes["/app"] = struct{}{}

	config.Volumes = volumes
	config.ExposedPorts = exposed_ports
	config.Cmd = []string{"/bin/bash", "-c",  cmd}
	config.Env = []string{"PORT=80"}
	config.Image = "heroku-runtime"
	config.WorkingDir = "/app"

	container, err := docker_client.CreateContainer(dockerClient.CreateContainerOptions{}, &config)
	if err != nil {
		panic("create container")
	}

	err = docker_client.StartContainer(container.ID, &docker.HostConfig{Binds: []string{cwd + "/app:/app"}})
	if err != nil {
		panic("start container")
	}
	fmt.Printf("%+s\n", container.Name)
}

func main() {
	app_name := "ignite-heroku"
	release_name := os.Args[3]
	slug_id := GetSlugIdForRelease(app_name, release_name)
	fmt.Println("Slug ID:", slug_id)

	slug := GetSlugBlobUrl(app_name, slug_id)
	fmt.Println("Slug URL:", slug.Blob["get"].(string))

	tar_buffer := FetchSlugArchive(app_name, slug.Blob["get"].(string))
	fmt.Println("Download complete, unpacking.")

	UntarFiles(tar_buffer)
	fmt.Println("Unpack complete")
	RunDockerContainer(slug.Process_Types[os.Args[4]].(string))

}
