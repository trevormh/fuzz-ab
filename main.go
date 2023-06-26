package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
)


func printResults(results Result) {
	for i := 1; i <= len(results.ExtractedResult); i ++ {
		fmt.Printf("Group %#v - Complete: %#v Failed: %#v Per Second: %#v\n",
		i,
		results.ExtractedResult[i-1]["complete"],
		results.ExtractedResult[i-1]["failed"],
		results.ExtractedResult[i-1]["per_second"],
		)
	}

	fmt.Println("") // blank line on purpose
	fmt.Println("Summary")
	fmt.Printf("Total Complete: %#v\n", results.Summary.Complete)
	fmt.Printf("Total Failed: %#v\n", results.Summary.Failed)
	fmt.Printf("Avg Requests Per Second: %#v\n", results.Summary.PerSecond)
}

func checkDirsExist() (error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return errors.New("Unable to obtain home directory")
	}

	path := home + "/fuzz-ab/tmp"

	fmt.Println()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, 0777)
		if err != nil {
			return errors.New(fmt.Sprintf("Unable to create directory at %#v", path))
		}
	}
	return nil
}


func main() {
	pathPtr := flag.String("path", "", "Path to file containing requests to build and send")
	verbosePtr := flag.Bool("v", false, "Verbose output")
	request_names := flag.Args() // optional args to specify which requests in json file to run
	flag.Parse()

	var path string
	if *pathPtr == "" {
		fmt.Println("path flag is required")
		return
	}
	path = *pathPtr

	verbose := false
	if *verbosePtr == true {
		verbose = true
	}

	err := checkDirsExist()
	if err != nil {
		fmt.Println(err)
		return
	}

	data, err := Import(path, request_names)
	if err != nil {
		fmt.Println(err)
		return
	}

	requests,err := BuildAbRequests(data)
	if err != nil {
		fmt.Println(err)
		return
	}

	results := ExecuteRequests(requests, verbose)

	printResults(results)
}