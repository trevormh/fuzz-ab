package main

import (
	"encoding/json"
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
	UrlVars map[string][]interface{} `json:"url-vars"`
	Payload map[string]interface{} `json:"body"`
	AbOptions map[string]interface{} `json:"ab-options"`
}

func get_file_contents(path string) ([]byte,error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return b, err
}


// Convert the json file contents to map
func unmarshall_json(data []byte, request_names []string) (map[string]JsonRequestBody, error) {
	var all_requests JsonRequest
	// Each key in the json object is an unknown name.
	// Unmarshall the json body to get all of them.
	if err := json.Unmarshal(data, &all_requests); err != nil {
		return nil, err
	}

	// Users can specify which requests they want to run when the 
	// json file contains more than one request.
	// If the user specifies the requests to run then discard
	// everything else not in the request_names slice
	requests := make(map[string]JsonRequestBody)
	for _,name := range request_names {
		if _, exists := all_requests.Name[name]; exists {
			requests[name] = all_requests.Name[name]
		}
	}

	// unmarshal the bodies of each request (type JsonRequestBody)
	if err := json.Unmarshal([]byte(data), &requests); err != nil {
		return nil, err
	}

	return requests, nil
}


func Import(request_names []string, path string) (map[string]JsonRequestBody, error) {
	file, err := get_file_contents(path)
	if err != nil {
		return nil, err
	}

	// convert the json data to a map
	data, err := unmarshall_json(file, request_names)
	if err != nil {
		return nil, err
	}

	return data, err
}