package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
)

type JsonRequest struct {
	Name map[string]JsonRequestBody `json:"-"`
}

type JsonRequestBody struct {
	Url string `json:"url"`
	Method string `json:"method"`
	NumRuns int `json:"num_runs"`
	NumPerRun int `json:"num_per_run"`
	Concurrent int `json:"concurrent"`
	UrlVars map[string]interface{} `json:"url-vars"`
	Payload map[string]interface{} `json:"body"`
	AbOptions []string `json:"ab-options"`
}

type Request struct {
	Method string `json:"method"`
	Payload map[string]interface{} `json:"body"`
	Requests [][]string
}


func get_file_contents(path string) ([]byte,error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, err
}


func parse_json(data []byte, request_names []string) (map[string]JsonRequestBody, error) {
	var all_requests JsonRequest
	// first unmarshall the parsed json contents to get all the request names
	if err := json.Unmarshal(data, &all_requests); err != nil {
		return nil, err
	}

	// If requests_names slice has contents only those requests should be kept.
	requests := make(map[string]JsonRequestBody)
	for _,name := range request_names {
		// Check if the name exists in all_requests and
		// add it to the new requests variable if it's found.
		if _, exists := all_requests.Name[name]; exists {
			requests[name] = all_requests.Name[name]
		}
	}

	// unmarshal the bodies of each request (JsonRequestBody)
	if err := json.Unmarshal([]byte(data), &requests); err != nil {
		return nil, err
	}

	return requests, nil
}


/*
Takes the parsed json data that has been converted to a map and
builds the requests that will be used to call ab
*/
func build_requests(request_data map[string]JsonRequestBody) ([]Request, error) {
	var requests []Request

	fmt.Println(request_data)

	return requests, nil
}


func GetRequests(request_names []string, path string) ([]Request, error) {
	var requests []Request
	
	// get the path of the json file
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return requests, errors.New("Unable to locate home directory")
		}
		path = home + "/fuzz-ab.json"
	}

	// read the file contents
	file, err := get_file_contents(path)
	if err != nil {
		return nil, err
	}

	// parse the json data
	parsed_req, err := parse_json(file, request_names)
	if err != nil {
		return nil, err
	}

	return build_requests(parsed_req)
}
