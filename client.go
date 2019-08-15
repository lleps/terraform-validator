package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
)

func main() {
	args := os.Args
	if len(args) != 3 {
		fmt.Println(args[0], "host", "plan-file")
		return
	}
	host := args[1]
	planFile := args[2]
	content, err := ioutil.ReadFile(planFile)
	if err != nil {
		log.Fatal("Can't read planfile:", err)
		return
	}

	asBase64 := base64.StdEncoding.EncodeToString(content)
	resp, err := http.Post(host+"/validate", "text/plain", strings.NewReader(asBase64))
	if err != nil {
		log.Fatal("Can't post to /validate:", err)
		return
	}

	if resp.StatusCode != http.StatusOK {
		log.Fatal("Bad HTTP code:", resp.StatusCode)
		return
	}

	bodyResp, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Can't parse response:", err)
		return
	}

	fmt.Println("Validation sent. Awaiting response...")
	log.Println(string(bodyResp))
}
