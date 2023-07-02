package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strings"
	"sync"

	"github.com/trevormh/go-cartesian-product-map"
)


type AbRequest struct {
	Requests [][]string
	Delay [2]int
	PayloadPaths []string
}


// regex pattern for variables which are chars enclosed in double braces
// Ex: https://www.{{DOMAIN}}.com/{{SLUG}}
const var_pattern = `{{(.*?)}}`


// Converts a string url into a slice of strings where each location
// of a variable is an empty index in the slice.
// Also builds and returns a map with each key being the variable name and
// its value is a slice of the indexes where the values should be replaced.
// Ex: https://www.example.com/{{some}}/test/{{var}}/{{some}}
// is turned into [https://www.example.com/, ,/test/, , ]
// and also returns map[some:[1,4] var:[3]]
func (request JsonRequestBody) extractUrlVars() ([]string, map[string][]int) {
	re := regexp.MustCompile(var_pattern)
	matches := re.FindAllStringIndex(request.Url, -1)

	var url_slice []string
	// var_map indicates where variables should be added to in url_slice
	var_map := make(map[string][]int)

	for i, match := range matches {
		var_name := request.Url[match[0]+2:match[1]-2] // +/- 2 for the double braces
		
		if len(url_slice) == 0 {
			// select the url segment leading up to the first brace's index
			url_slice = append(url_slice, request.Url[:match[0]])
		} else {
			// append segment between the last match ending index and starting index of current match
			url_slice = append(url_slice, request.Url[matches[i-1][1]:match[0]])
		}
		url_slice = append(url_slice, "") // empty placeholder where variable will be swapped

		if _, exists := var_map[var_name]; exists {
			var_map[var_name] = append(var_map[var_name], len(url_slice)-1)
		} else {
			var_map[var_name] = []int{len(url_slice)-1}
		}
	}
	return url_slice, var_map
}

// swaps variables in a request payload/body with its value
func replacePayloadVars(payload map[string]interface{},  payload_vars map[string]interface{}) {
	var wg sync.WaitGroup
	wg.Add(1)
	iteratePayload(&wg, payload, payload_vars)
	wg.Wait();
}

// Recursively iterates over the payload request body and
// substitutes variables for their respsective values
func iteratePayload(wg *sync.WaitGroup, payload map[string]interface{}, payload_vars map[string]interface{}) {
	defer wg.Done()

	for key, val := range payload {
		val_type := reflect.TypeOf(val).String()
		if val_type == "string" || val_type == "float64" {
			regexSwapValues(key, val, payload, payload_vars)
		} else {
			// val is either an object or array that needs to be iterated over
			if val_map, ok := val.(map[string]interface{}); ok {
				for key, val := range val_map {
					v_type := reflect.TypeOf(val).String()
					// current index value is another object/array...iterate again
					if value, ok := val.(map[string]interface{}); ok {
						wg.Add(1)
						go iteratePayload(wg, value, payload_vars)
					// current index value can be assigned a value
					} else if v_type == "string" || v_type == "float64" {
						regexSwapValues(key, val, val_map, payload_vars)
					} else {
						fmt.Println("Unable to identify payload field type", key, val)
					}
				}
			}
		}
	}
}

// ab requires that POST/PUT payloads are saved in a file
func saveTmpPayload(payload map[string]interface{}, i int) (string) {
	json_str, err := json.Marshal(payload)
	if err != nil{
		panic("Unable to convert to JSON")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic("Unable to locate home directory")
	}
	path := home + "/fuzz-ab/tmp/" + ConvertToStr(i) + ".json"

    f, err := os.Create(path)
    if err != nil {
        panic(err)
    }

	f.Write(json_str)

    defer f.Close()

	return path
}


//Swaps variables in a string (identified by double curly braces)
// with a value from a map of variables, keyed by the variable's name.
func regexSwapValues(key string, val interface{}, property map[string]interface{}, payload_vars map[string]interface{}) {
	val_str := fmt.Sprintf("%v", val) 
	re := regexp.MustCompile(var_pattern)
	matches := re.FindAllStringIndex(val_str, -1)
	if len(matches) > 0 {
		for _, match := range matches {
			// +/- 2 to account for the braces
			var_name := val_str[match[0]+2:match[1]-2] 
			property[key] = payload_vars[var_name]
		}
	}
}

/*
Get cartesian product (aka all combinations) for a map of slices.
Ex: input = {"key1": [1,2], "key2": ["a", "b"]}
Result: [{"key1":1, "key2":"a"}, {"key1":1 "key2":"b"},
	{"key1":2, "key2":"a"}, {"key1":2, "key2":"b"}]
*/
func (request JsonRequestBody) getVarCombinations(property map[string][]interface{}) ([]map[string]interface{}) {
	var combinations []map[string]interface{}

	for combo := range cartesian.Iter(property) {
		combinations = append(combinations, combo)
	}
	return combinations
}

// Takes the variables extracted from the JSON input file
// and inserts them into the url_slice in their positions
// according to the var_locations map and sets the URL
// property on the JsonRequestBody request.
func (request *JsonRequestBody) setUrl(url_slice []string, var_locations map[string][]int, vars map[string]interface{}) {
	for var_name, val := range vars {
		if indices, ok := var_locations[var_name]; ok {
			// One variable can be used in multiple locations so loop over everything...
			for _, idx := range indices {
				url_slice[idx] = ConvertToStr(val)
			}
		}
	}
	request.Url = strings.Join(url_slice, "")
}

// combines the url, payload and ab options to create an ab request
func (ab *AbRequest) createAbRequest(request JsonRequestBody) {
	var req_params []string
	for key, val := range request.AbOptions {
		req_params = append(req_params, key)
		req_params = append(req_params, ConvertToStr(val))
	}

	if len(ab.PayloadPaths) > 0 && (request.Method == "POST" || request.Method == "PUT") {
		var payload_flag string
		if request.Method == "POST" {
			payload_flag = "-p"
		} else if request.Method == "PUT" {
			payload_flag = "-u"
		}

		for _, path := range ab.PayloadPaths {
			options := append(req_params, payload_flag, path, request.Url)
			ab.Requests = append(ab.Requests, options)
		}
	} else {
		options := append(req_params, request.Url)
		ab.Requests = append(ab.Requests, options)
	}
}

func (ab *AbRequest) setDelay(delay interface{}) {
	switch delay.(type) {
	case nil:
		ab.Delay = [2]int{0, 0}
	case float64:
		int_val := ConvertToInt(delay)
		ab.Delay = [2]int{int_val, int_val}
	case []interface{}:
		var int_vals [2]int
		for i, v := range delay.([]interface{}) {
			int_vals[i] = ConvertToInt(v)
		}
		ab.Delay = [2]int{int_vals[0], int_vals[1]}
	default:
		fmt.Println("Unable to set delay type. Must be int or array of ints of length 2. Defaulting to no delay.")
		ab.Delay = [2]int{0, 0}
	}
}


// Builds map of requests into ab calls/requests
func BuildAbRequests(request_data map[string]JsonRequestBody) ([]AbRequest, error) {
	var ab_requests []AbRequest

	for name, request := range request_data {
		var ab AbRequest

		// Find any variables in the payload, get all the combinations
		// and swap the variable values 
		if len(request.PayloadVars) > 0 && len(request.Payload) > 0 {
			payload_combos := request.getVarCombinations(request.PayloadVars)
			for i, combo := range payload_combos {
				payload_copy := CopyMap(request_data[name].Payload)
				replacePayloadVars(payload_copy, combo)
				path := saveTmpPayload(payload_copy, i)
				ab.PayloadPaths = append(ab.PayloadPaths, path)
			}
		}

		url_slice, url_var_map := request_data[name].extractUrlVars()
		// Get all the combinations of parameters provided in the json file and build the ab_calls
		url_var_combos := request.getVarCombinations(request.UrlVars)
		for _, combo := range url_var_combos {
			request.setUrl(url_slice, url_var_map, combo)
			ab.createAbRequest(request)
		}

		ab.setDelay(request.Delay)

		ab_requests = append(ab_requests, ab)
	}

	return ab_requests, nil
}