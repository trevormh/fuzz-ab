package main

import (
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/trevormh/go-cartesian-product-map"
)

type AbRequest struct {
	Method string
	Payload map[string]interface{}
	NumRuns int
	Requests []string
}


// Converts the url into a slice of strings where each location
// of a variable in the url string is an empty index in the slice.
// Also builds a map with each key being the variable name and
// its value is a slice of the indexes where the values should
// be replaced in the sliced url.
// Ex: https://www.example.com/{{some}}/test/{{var}}
// is turned into [https://www.example.com/, ,/test/, ]
func (request JsonRequestBody) extract_url_vars() ([]string, map[string][]int) {
	// regex to find all the starting and ending 
	// indicse of variable locations in slice
	re := regexp.MustCompile(`{{(.*?)}}`)
	matches := re.FindAllStringIndex(request.Url, -1)

	var url_slice []string
	var_map := make(map[string][]int)

	// iterate through the matches and build up the url_slices
	for i, match := range matches {
		// add 2 to the start and subtract 2 from the end to offset the braces
		var_name := request.Url[match[0]+2:match[1]-2] 
		
		// select the parts of the url leading up to the first brace's index
		if len(url_slice) == 0 {
			url_slice = append(url_slice, request.Url[:match[0]])
		} else {
			// append segment between the last match ending index and starting index of current match
			url_slice = append(url_slice, request.Url[matches[i-1][1]:match[0]])
		}
		url_slice = append(url_slice, "") // empty placeholder index where variable will be swapped

		// add an entry to the var_map indicating which indices
		// variables should be added to in url_slice
		if _, exists := var_map[var_name]; exists {
			var_map[var_name] = append(var_map[var_name], len(url_slice)-1)
		} else {
			var_map[var_name] = []int{len(url_slice)-1}
		}
	}
	return url_slice, var_map
}

/*
Get cartesian product (aka all combinations) for a map of slices.
Ex: input = {"key1": [1,2], "key2": ["a", "b"]}
Result: [{"key1":1, "key2":"a"}, {"key1":1 "key2":"b"},
	{"key1":2, "key2":"a"}, {"key1":2, "key2":"b"}]
*/
func (request JsonRequestBody) get_var_combinations() ([]map[string]interface{}) {
	var combinations []map[string]interface{}

	for combo := range cartesian.Iter(request.UrlVars) {
		combinations = append(combinations, combo)
	}
	return combinations
}

// Takes the variables extracted from the JSON input file
// and inserts them into the url_slice in their positions
// according to the var_locations map and sets the URL property 
// on the JsonRequestBody request.
func (request *JsonRequestBody) replace_url_vars(url_slice []string, var_locations map[string][]int, vars map[string]interface{}) {
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
	request.Url = strings.Join(url_slice, "")
}

// combines the url and ab options to 
func (request JsonRequestBody) create_ab_request() string {
	var ab_options []string
	// iterate over the provided AbOptions and remove concurrent or 
	// number of requests if provided
	for _, option := range request.AbOptions {
		// check for concurrent flag and skip if it's provided
		concurrent_match, _ := regexp.Match(`-c \d+`, []byte(option))
		if concurrent_match {
			continue
		}
		num_match, _ := regexp.Match(`-n \d+`, []byte(option))
		if num_match {
			continue
		}
		ab_options = append(ab_options, option)
	}

	return "ab " + strings.Join(ab_options, " ") + " " + request.Url
}


// Receives a map of request data and builds the ab requests that will be used to call ab
func BuildAbRequests(request_data map[string]JsonRequestBody) ([]AbRequest, error) {
	var ab_requests []AbRequest

	for name, request := range request_data {
		url_slice, var_map := request_data[name].extract_url_vars()

		ab_request := AbRequest {
			Method: request.Method,
			NumRuns: request.NumPerRun,
			Payload: request.Payload,
		}

		// Get all the combinations of parameters provided in the json file
		var_combos := request.get_var_combinations()
		// Build the ab_calls using these combinations
		for _, combo := range var_combos {
			request.replace_url_vars(url_slice, var_map, combo)
			ab_request.Requests = append(ab_request.Requests, request.create_ab_request())
			
		}
		ab_requests = append(ab_requests, ab_request)
	}
	return ab_requests, nil
}