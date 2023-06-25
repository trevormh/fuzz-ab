package main

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/trevormh/go-cartesian-product-map"
)

type AbRequest struct {
	Requests [][]string
}

// Regex to identify variables in a string
// Ex: https://www.{{DOMAIN}}.com/{{SLUG}}
const braces_regex = `{{(.*?)}}`


// Converts a string url into a slice of strings where each location
// of a variable is an empty index in the slice.
// Also builds and returns a map with each key being the variable name and
// its value is a slice of the indexes where the values should be replaced.
// Ex: https://www.example.com/{{some}}/test/{{var}}/{{some}}
// is turned into [https://www.example.com/, ,/test/, , ]
// and also returns map[some:[1,4] var:[3]]
func (request JsonRequestBody) extract_url_vars() ([]string, map[string][]int) {
	// Find any variables in the url
	re := regexp.MustCompile(braces_regex)
	matches := re.FindAllStringIndex(request.Url, -1)

	var url_slice []string
	var_map := make(map[string][]int)

	for i, match := range matches {
		var_name := request.Url[match[0]+2:match[1]-2] // +/- 2 for the double braces
		
		// select the parts of the url leading up to the first brace's index
		if len(url_slice) == 0 {
			url_slice = append(url_slice, request.Url[:match[0]])
		} else {
			// append segment between the last match ending index and starting index of current match
			url_slice = append(url_slice, request.Url[matches[i-1][1]:match[0]])
		}
		url_slice = append(url_slice, "") // empty placeholder where variable will be swapped

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

// swaps variables in a request payload/body with its value
func replace_payload_vars(payload map[string]interface{},  payload_vars map[string]interface{}) {
	var wg sync.WaitGroup
	// fmt.Println(payload_vars)
	wg.Add(1)
	iterate_payload(&wg, payload, payload_vars)
	wg.Wait();
}


/*
Recursively iterates over the payload request body and
substitutes variables for their respsective values
*/
func iterate_payload(wg *sync.WaitGroup, payload map[string]interface{}, payload_vars map[string]interface{}) {
	defer wg.Done()

	for key, val := range payload {
		val_type := reflect.TypeOf(val).String()
		if val_type == "string" || val_type == "float64" {
			regex_swap_values(key, val, payload, payload_vars)
		} else {
			// val is either an object or array that needs to be iterated over
			if val_map, ok := val.(map[string]interface{}); ok {
				for key, val := range val_map {
					v_type := reflect.TypeOf(val).String()
					// current index value is another object/array...iterate again
					if value, ok := val.(map[string]interface{}); ok {
						wg.Add(1)
						go iterate_payload(wg, value, payload_vars)
					// current index value can be assigned a value
					} else if v_type == "string" || v_type == "float64" {
						regex_swap_values(key, val, val_map, payload_vars)
					} else {
						fmt.Println("Unable to identify payload field type", key, val)
					}
				}
			}
		}
	}
}

func save_tmp_payload(payload map[string]interface{}, i int) (string) {
	json_str, err := json.Marshal(payload)
	if err != nil{
		panic("Unable to convert to JSON")
	}

	home, err := os.UserHomeDir()
	if err != nil {
		panic("Unable to locate home directory")
	}
	path := home + "/fuzz-ab/tmp/" + convert_to_str(i) + ".json"

    f, err := os.Create(path)
    if err != nil {
        panic(err)
    }

	f.Write(json_str)

    defer f.Close()

	return path
}

/*
Swaps variables in a string (identified by double curly braces)
with a value from a map of variables, keyed by the variable's name.
*/
func regex_swap_values(key string, val interface{}, property map[string]interface{}, payload_vars map[string]interface{}) {
	val_str := fmt.Sprintf("%v", val) 
	re := regexp.MustCompile(braces_regex)
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
func (request JsonRequestBody) get_var_combinations(property map[string][]interface{}) ([]map[string]interface{}) {
	var combinations []map[string]interface{}

	for combo := range cartesian.Iter(property) {
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
	} else if intval, ok := val.(int); ok {
		return strconv.Itoa(intval)
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

// combines the url, payload and ab options to create an ab request
func (request JsonRequestBody) create_ab_request(payload_path string) []string {
	var ab_options []string

	for key, val := range request.AbOptions {
		ab_options = append(ab_options, key)
		ab_options = append(ab_options, convert_to_str(val))
	}

	if len(payload_path) > 0 && (request.Method == "POST" || request.Method == "PUT") {
		var payload_flag string
		if request.Method == "POST" {
			payload_flag = "-p"
		} else if request.Method == "PUT" {
			payload_flag = "-u"
		}
			
		ab_options = append(ab_options, payload_flag, payload_path)
	}

	return append(ab_options, request.Url)
}

func CopyMap(m map[string]interface{}) map[string]interface{} {
    cp := make(map[string]interface{})
    for k, v := range m {
        vm, ok := v.(map[string]interface{})
        if ok {
            cp[k] = CopyMap(vm)
        } else {
            cp[k] = v
        }
    }
    return cp
}


// Builds map of requests into ab calls/requests
func BuildAbRequests(request_data map[string]JsonRequestBody) ([]AbRequest, error) {
	var ab_requests []AbRequest

	for name, request := range request_data {
		var ab AbRequest

		// Find any variables in the payload and get all the combinations
		var payloads []string
		if len(request.PayloadVars) > 0 && len(request.Payload) > 0 {
			payload_combos := request.get_var_combinations(request.PayloadVars)
			for i, combo := range payload_combos {
				payload_copy := CopyMap(request_data[name].Payload)
				replace_payload_vars(payload_copy, combo)
				path := save_tmp_payload(payload_copy, i)
				payloads = append(payloads, path)
			}
		}

		// find any variables in the URL
		url_slice, url_var_map := request_data[name].extract_url_vars()

		// Get all the combinations of parameters provided in the json file and build the ab_calls
		url_var_combos := request.get_var_combinations(request.UrlVars)
		for _, combo := range url_var_combos {
			request.replace_url_vars(url_slice, url_var_map, combo)
			if len(payloads) > 0 {
				for _, path := range payloads {
					ab.Requests = append(ab.Requests, request.create_ab_request(path))
				}
			} else {
				ab.Requests = append(ab.Requests, request.create_ab_request(""))
			}
		}
		ab_requests = append(ab_requests, ab)
	}

	return ab_requests, nil
}