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
	Requests [][]string
}


// Converts the url into a slice of strings where each location
// of a variable in the url string is an empty index in the slice.
// Also builds a map with each key being the variable name and
// its value is a slice of the indexes where the values should
// be replaced in the sliced url.
// Ex: https://www.example.com/{{some}}/test/{{var}}
// is turned into [https://www.example.com/, ,/test/, ]
func (request JsonRequestBody) extract_url_vars() ([]string, map[string][]int) {
	// Regex to locate the variables, which are words surrounded
	// by doubly braces. Ex: {{VARIABLE_NAME}}
	re := regexp.MustCompile(`{{(.*?)}}`)
	matches := re.FindAllStringIndex(request.Url, -1)

	var url_slice []string
	var_map := make(map[string][]int)

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

func convert_to_str(val interface{}) (string) {
	var str_val string
	// value is already a string
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
	return str_val
}

// Takes the variables extracted from the JSON input file
// and inserts them into the url_slice in their positions
// according to the var_locations map. Using this data also
// sets the URL property on the JsonRequestBody request.
func (request *JsonRequestBody) replace_url_vars(url_slice []string, var_locations map[string][]int, vars map[string]interface{}) {
	for var_name, val := range vars {
		// check that the variable is actually used in the url.
		if indices, ok := var_locations[var_name]; ok {
			// One variable can be used in multiple locations so loop over everything...
			for _, idx := range indices {
				url_slice[idx] = convert_to_str(val)
			}
		}
	}
	request.Url = strings.Join(url_slice, "")
}

// combines the url and ab options to create an ab request
func (request JsonRequestBody) create_ab_request() []string {
	var ab_options []string

	for key, val := range request.AbOptions {
		ab_options = append(ab_options, key)
		ab_options = append(ab_options, convert_to_str(val))
	}
	return append(ab_options, request.Url)
}


// Builds map of requests into ab calls/requests
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