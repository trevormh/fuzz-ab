package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/trevormh/go-cartesian-product-map"
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
	AbOptions []string `json:"ab-options"`
}

type Request struct {
	Method string
	Payload map[string]interface{}
	NumRuns int
	Requests []string
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

	// unmarshal the bodies of each request (JsonRequestBody)
	if err := json.Unmarshal([]byte(data), &requests); err != nil {
		return nil, err
	}

	return requests, nil
}

// Converts the url into a slice of strings where each location
// of a variable in the url string is an empty index in the slice.
// Ex: https://www.example.com/{{some}}/test/{{var}}
// is turned into [https://www.example.com/, ,/test/, ]
// Also builds a map with each key being the variable name and
// its value is a slice of the indexes where the values should
// be replaced in the sliced url.
func parse_url_vars(url string) ([]string, map[string][]int) {
	// regex to find all the starting and ending 
	// indicse of variable locations in slice
	re := regexp.MustCompile(`{{(.*?)}}`)
	matches := re.FindAllStringIndex(url, -1)
	
	// fmt.Println(matches)
	// panic("something")

	var url_slice []string
	var_map := make(map[string][]int)
	
	
	idx := 0 // tracks the indices for variable placement in url_slice
	// iterate through the matches and build up the url_slices
	for i, match := range matches {
		// add 2 to the start and subtract 2 from the end to offset the braces
		var_name := url[match[0]+2:match[1]-2] 
		
		// select the parts of the url leading up to the first brace's index
		if idx == 0 {
			url_slice = append(url_slice, url[:match[0]])
			idx += 1
		} else {
			// need to match between the last match ending index and starting index of current match
			url_slice = append(url_slice, url[matches[i-1][1]:match[0]])
			idx += 1
		}
		url_slice = append(url_slice, "") // empty placeholder index where variable will be swapped

		// add an entry to the var_map indicating which indices
		// variables should be added to in url_slice
		if _, exists := var_map[var_name]; exists {
			var_map[var_name] = append(var_map[var_name], idx)
		} else {
			var_map[var_name] = []int{idx}
		}
		idx += 1
	}
	return url_slice, var_map
}

/*
Makes cartesian product (aka all combinations) for a 
map of slices.
Ex:
input = {"key1": [1,2], "key2": ["a", "b"]}
Result: [
	{"key1":1, "key2":"a"}, {"key1":1 "key2":"b"},
	{"key1":2, "key2":"a"}, {"key1":2, "key2":"b"},
]
*/
func get_var_combinations(vars map[string][]interface{}) ([]map[string]interface{}) {
	var combinations []map[string]interface{}

	for combo := range cartesian.Iter(vars) {
		combinations = append(combinations, combo)
	}
	return combinations
}

// Takes the variables extracted from the JSON input file
// and inserts them into the url_slice in their positions
// according to the var_locations map and returns it as a string.
func replace_url_vars(url_slice []string, var_locations map[string][]int, vars map[string]interface{}) (string) {
	// vars are the variables in the url to be replaced where each
	// key is the var name and the value is the value to be inserted
	for var_name, val := range vars {
		// check that the variable is actually used in the url.
		// idxs are the locations in the url_slice where the
		// value should be placed.
		if indices, exists := var_locations[var_name]; exists {
			var str_val string

			// value is a string
			if str, ok := val.(string); ok {
				str_val = str
			// value is float64, cast it to a string					
			} else if float_val, ok := val.(float64); ok {
				str_val = strconv.FormatFloat(float_val, 'f', -1, 64)
			} else {
				fmt.Println(val)
				fmt.Println(reflect.TypeOf(val))
				panic("Value could not be cast to string")
			}
			// One variable can be used in multiple locations
			for _, idx := range indices {
				url_slice[idx] = str_val
			}
		}
	}
	return strings.Join(url_slice, "")
}

/*
Takes the parsed json data that has been converted to a map and
builds the requests that will be used to call ab
*/
func build(request_data map[string]JsonRequestBody) ([]Request, error) {
	var requests []Request

	for name, req_data := range request_data {
		url_slice, var_map := parse_url_vars(request_data[name].Url)

		var request Request
		request.Method = req_data.Method
		request.NumRuns = req_data.NumPerRun
		request.Payload = req_data.Payload

		var url_strs []string
		var_combos := get_var_combinations(req_data.UrlVars)
		for _, combo := range var_combos {
			url := replace_url_vars(url_slice, var_map, combo)
			// each combo will be added the number times specified by the num_per_run
			for i := 0; i < req_data.NumPerRun; i++ {
				url_strs = append(url_strs, url)
			}
		}
		request.Requests = url_strs
		requests = append(requests, request)
	}
	return requests, nil
}


func BuildRequests(request_names []string, path string) ([]Request, error) {
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

	// convert the json data to a map
	request, err := unmarshall_json(file, request_names)
	if err != nil {
		return nil, err
	}

	return build(request)
}
