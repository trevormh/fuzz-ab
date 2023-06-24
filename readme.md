# Fuzz-AB

## What is it?
This is a wrapper around [Apache Bench](https://httpd.apache.org/docs/2.4/programs/ab.html) written in Go to provide it with some basic [fuzz testing](https://en.wikipedia.org/wiki/Fuzzing) capabilities. Create a JSON file with requests you wish to send (see the sample section), add some values you wish to use for fuzz testing and fuzz-ab will build cartesian products (aka combinations) of all the request variables, turn them into ab requests and execute all of them.

### For Example...
With the url `https://www.example.com/{{variable1}}/?param={{variable2}}`, if you wish to test multiple values for the variables, strings surrounded by `{{ }}`, you can construct an array of options such as `["a","b"]` for `variable1` and `[1,2]` for `variable2`. All combinations of those values will be constructed into URLs and made into AB calls

`https://www.example.com/a/?param=1`

`https://www.example.com/a/?param=2`

`https://www.example.com/b/?param=1`

`https://www.example.com/b/?param=2`


## How to use it

Note: ab is required to be installed on your computer this to work. 

1. Create a JSON request file (see JSON Input File section below) and name it whatever you like.
2. Build with `go build`
3. Run with `./fuzz-ab -path=/path/to/requests.json req1 req2 req3`

Where `req1 req2 ....` are optional key names of the requests you wish to run in your requests JSON file. If none of the request name arguments are provided then every key and corresponding request data found in the JSON file will be run. 

## JSON Input File

The input file is how fuzz-ab requests are defined. The file must contain a single JSON object where each key is the name of a request its value is an object consisting of the following params:

`url` [string - required]. URL to send request to. Can contain variables specified by strings surrounded by double braces (Ex: `{{VARIABLE_NAME}}`) whose values are provided in the `url-vars` property.

`url-vars` [object - optional]. JSON object to define values for variables in the URL. Keys are the variable names, values can be any valid JSON.

`method` [string - optional]. HTTP method for the request. Defaults to `GET` if not specified.

`ab-options` [object - optional]. JSON object to define [apache bench](https://httpd.apache.org/docs/2.4/programs/ab.html) paramaters. Ex: `"ab-options": {"-c": 10}`

`payload` [object - optional]. Payload/body for requests in the form of a JSON object. Can contain variables.

`payload-vars` [object - optional]. JSON object to define values for variables in the payload. Keys are the variable names, values can be any valid JSON.


## Sample Input File

```
This example defines 2 requests, one named req1 and a second named req2.

{
    "req1": {
        "url": "https://www.something.com/route/{{slug_var}}/?some_param={{param_var}}&something={{param_var}}",
        "method": "GET",
        "url-vars": {
            "slug_var": ["value1", "value2", "value3"],
            "param_var": [1,2,3,4,5,6,7,8,9,0]
        },
        "ab-options": {
            "-n": 10,
            "-c": 5,
            "-C" : "session=123abc",
            "-H": "csrftoken: abc123"
        }
    },
    "req2": {
        "url": "https://www.something.com/{{some_slug}}",
        "method": "POST",
        "url-vars": {
            "some_slug": ["postslug1", "postslug2"]
        },
        "payload": {
            "id": 5,
            "some_values": [1,2,3,4,5],
            "some_param": "{{some_param_value}}",
            "another_param": {
                "property1": 123,
                "property2": "{{property2_value}}"
            }
        },
        "payload-vars": {
            "some_param_value": ["value1", "value2", "value3"],
            "property2_value": [1,2,3,4,5]
        },
        "ab-options": {
            "-n": 5,
            "-c": 5,
            "-H": "csrftoken: token123"
        }
    }
}

```

The above example will be used to generate 60 ab requests between both `req1` and `req2`. 

`req1` has 2 variables: `slug_var` which has 3 values and `param_var` which has 10 for a total of 30 combinations

```
ab -n 10 -c 5 -C "session=123abc" -H "csrftoken: abc123" https://www.something.com/route/value1/?some_param=1&something=1
ab -n 10 -c 5 -C "session=123abc" -H "csrftoken: abc123" https://www.something.com/route/value1/?some_param=2&something=2
...
ab -n 10 -c 5 -C "session=123abc" -H "csrftoken: abc123" https://www.something.com/route/value3/?some_param=0&something=0
```


`req2` has 3 variables: `some_slug` in the url which has 2 values, `some_param_value` in the payload which has 3 values and `property2_value` also in the payload which has 5 values, for a total of 30 combinations (2 * 3 * 5). ab requires payloads to be in a file which it reads from, so fuzz-ab will create json files for all of these payloads in the /tmp/fuzz-ab directory for each run and removes them immediately after running.

```
ab -n 5 -c 2 -H "csrftoken: token123" -T application/x-www-form-urlencoded -p /tmp/fuzz-ab/payload1.json https://www.something.com/postslug1
ab -n 5 -c 2 -H "csrftoken: token123" -T application/x-www-form-urlencoded -p /tmp/fuzz-ab/payload2.json https://www.something.com/postslug1
...
ab -n 5 -c 2 -H "csrftoken: token123" -T application/x-www-form-urlencoded -p /tmp/fuzz-ab/payload30.json https://www.something.com/postslug2
```