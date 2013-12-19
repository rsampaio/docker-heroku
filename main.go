package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

var base_url = "https://api.heroku.com"

type Releases struct {
	Id   string
	Slug map[string]interface{}
}

func HttpClient() *http.Client {
	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{},
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}
	return client
}

func MakeReleaseRequest(app_name string, release string) *http.Request {
	request_url := fmt.Sprintf("%s/apps/%s/releases/%s", base_url, app_name, release)
	request, err := http.NewRequest("GET", request_url, nil)
	if err != nil {
		panic("request create")
	}
	request.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	request.SetBasicAuth(os.Args[1], os.Args[2])
	return request
}

func GetSlugForRelease(app_name string, release_name string) string {
	client := HttpClient()
	request := MakeReleaseRequest(app_name, release_name)
	resp, err := client.Do(request)
	body, err := ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	var release Releases
	err = json.Unmarshal(body, &release)
	if err != nil {
		panic(err)
	}
	return release.Slug["id"].(string)
}

func main() {
	app_name := "ignite-heroku"
	release_name := "3"
	fmt.Println(GetSlugForRelease(app_name, release_name))

}
