package main

import (
	"io/ioutil"
	"log"
	"net/http"
)

func main() {
	resp, _ := http.Get("http://localhost:8080/features")
	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
}