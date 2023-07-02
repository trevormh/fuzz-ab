package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

type JsonRequest struct {
	Name map[string]JsonRequestBody `json:"-"`
}

type JsonRequestBody struct {
	Url string `json:"url"`
	Method string `json:"method"`
	Delay interface{} `json:"delay"`
	UrlVars map[string][]interface{} `json:"url-vars"`
	Payload map[string]interface{} `json:"payload"`
    PayloadVars map[string][]interface{} `json:"payload-vars"`
	AbOptions map[string]interface{} `json:"ab-options"`
}



// Convert the json file contents to map
func unmarshallJson(data []byte, request_names []string) (map[string]JsonRequestBody) {
    // decode the request names from the json file
	json_data := make(map[string]json.RawMessage)
    err := json.Unmarshal(data, &json_data)
    if err != nil {
		fmt.Println(err)
		panic("Error decoding JSON request file")
    }

	requests := make(map[string]JsonRequestBody)

	// If request names were passed in only decode the ones specified
	if len(request_names) > 0 {
		for _, req_name := range request_names {
			if _, exists := json_data[req_name]; exists {
				var body JsonRequestBody
				if err := json.Unmarshal(json_data[req_name], &body); err != nil {
					fmt.Println(err)
					panic("Error decoding JSON request file")
				}
				requests[req_name] = body
			}
		}
		if len(requests) == 0 {
			fmt.Println("No requests in JSON file matched any of the names provided: " + strings.Join(request_names, ","))
		}
	} else {
		// decode the entire json object
		for name, request := range json_data  {
			var body JsonRequestBody
			if err := json.Unmarshal(request, &body); err != nil {
				fmt.Println(err)
				panic("Error decoding JSON request file")
			}
			requests[name] = body
		}
	}

	return requests
}

func Import(path string,request_names []string) (map[string]JsonRequestBody, error) {
	file, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	data := unmarshallJson(file, request_names)

	return data, err
}