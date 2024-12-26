package http_client

import (
	"bytes"
	"io/ioutil"
	"net/http"
)

func PostJsonRequest(requestBody, url string) (response []byte, err error) {
	var jsonStr = []byte(requestBody)
	request, err := http.NewRequest(Post, url, bytes.NewBuffer(jsonStr))
	request.Header.Add(ContentType, Json)
	client := &http.Client{}
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}

func GetJsonRequest(requestBody, url string) (response []byte, err error) {
	var jsonStr = []byte(requestBody)
	request, err := http.NewRequest(Get, url, bytes.NewBuffer(jsonStr))
	request.Header.Add(ContentType, Json)
	client := &http.Client{}
	resp, err := client.Do(request)
	defer resp.Body.Close()
	if err != nil {
		return nil, err
	}
	body, _ := ioutil.ReadAll(resp.Body)
	return body, nil
}
