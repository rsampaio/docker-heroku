package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
)

type Releases struct {
	Id   string
	Slug map[string]interface{}
}

func main() {
	base_url := "https://api.heroku.com"
	app_name := "ignite-heroku"

	tr := &http.Transport{
		TLSClientConfig:    &tls.Config{},
		DisableCompression: true,
	}

	client := &http.Client{Transport: tr}

	request_url := fmt.Sprintf("%s/apps/%s/releases", base_url, app_name)
	request, err := http.NewRequest("GET", request_url, nil)
	if err != nil {
		panic("request create")
	}

	request.Header.Add("Accept", "application/vnd.heroku+json; version=3")
	request.SetBasicAuth(os.Args[1], os.Args[2])

	resp, err := client.Do(request)
	body, err := ioutil.ReadAll(resp.Body)

	defer resp.Body.Close()

	var release []Releases
	err = json.Unmarshal(body, &release)
	if err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", release)

}
