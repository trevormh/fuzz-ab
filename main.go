package main

import (
	"flag"
	"fmt"
	"os"
)



func main() {
	request_names := os.Args[1:] // specify which requests in json file to run
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

	// import json from file and convert to map
	data, err := Import(request_names, path)
	if err != nil {
		fmt.Println(err)
		return
	}

	// build the ab requests from the imported data
	requests,err := BuildAbRequests(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	Execute(requests)
}