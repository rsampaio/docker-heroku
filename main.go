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
)

var base_url = "https://api.heroku.com"

type Releases struct {
	Id   string
	Slug map[string]interface{}
}

type Slug struct {
	Blob         map[string]interface{}
	Process_type map[string]interface{}
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

func GetSlugBlobUrl(app_name string, slug_id string) string {
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
	return slug.Blob["get"].(string)
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
		fmt.Printf("Contents of %s:\n", hdr.Name)
		fmt.Printf("root dir: %s\n", path.Dir(hdr.Name))
		os.MkdirAll(path.Dir(hdr.Name), os.FileMode(hdr.Mode))
		file_content, _ := ioutil.ReadAll(tr)
		ioutil.WriteFile(hdr.Name, file_content, os.FileMode(hdr.Mode))
	}
}

func main() {
	app_name := "ignite-heroku"
	release_name := os.Args[3]
	slug_id := GetSlugIdForRelease(app_name, release_name)
	fmt.Println("Slug ID: ", slug_id)
	slug_url := GetSlugBlobUrl(app_name, slug_id)
	fmt.Println("Slug URL: ", slug_url)
	tar_buffer := FetchSlugArchive(app_name, slug_url)
	UntarFiles(tar_buffer)
}
