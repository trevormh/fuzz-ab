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

	requests,err := BuildRequests(request_names,*pathPtr)

	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println(requests)
}