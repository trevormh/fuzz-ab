package main

import (
	"flag"
	"fmt"
	"os"
)



func main() {
	req_names := os.Args[1:] // specify which requests in json file to run
	pathPtr := flag.String("path", "", "Path to file containing requests to build and send")
	flag.Parse()

	// set the path of the json file
	var path string
	if *pathPtr == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			panic("Unable to locate home directory")
		}
		path = home + "/fuzz-ab.json"
	} else {
		path = *pathPtr
	}

	requests,err := BuildRequests(req_names,path)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(requests)
}