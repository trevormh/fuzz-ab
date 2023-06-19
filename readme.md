# Fuzz-AB

This is a work in progress. Requests with payloads are not yet fully supported, but they will be soon!

## What is it?
This is a wrapper around [Apache Bench](https://httpd.apache.org/docs/2.4/programs/ab.html) written in Go to provide it with some basic [fuzz testing](https://en.wikipedia.org/wiki/Fuzzing) capabilities. Create a JSON file with requests you wish to send (see the sample section), add some values you wish to use for fuzz testing and fuzz-ab will build cartesian products (aka combinations) of all the request variables, turn them into ab requests and execute all of them.

###For Example...
With the url `https://www.example.com/{{variable1}}/?param={{variable2}}`, if you wish to test multiple values for the variables you can construct an array of options, such as `["a","b"]` for `variable1` and `[1,2]` for `variable2`, all combinations of those values will be constructed into URLs and made into AB calls

`https://www.example.com/a/?param=1`

`https://www.example.com/a/?param=2`

`https://www.example.com/b/?param=1`

`https://www.example.com/b/?param=2`


## How to use it

Note: AB is required to be installed on your computer this to work. 

1. Create a JSON request file (see sample for structure) and name it whatever you like.
2. Build with `go build`
3. Run with `./fuzz-ab -path=/path/to/requests.json req1 req2 req3`

Where `req1 req2 ....` are optional and names of the requests you wish to run in your requests JSON file. If none of the request name arguments are provided then every argument found in the JSON file will be run. 

## Sample JSON Input File

Variables can be used in the request URL and body/payload by wrapping the variable name in double curly braces (ex: `{{VAR_NAME}}`), and then assigning it a value in its respective section. i.e. `url` variables should be added to the `url-vars` section and `body`/payload variables should be added to the `body-vars` section.

You can name the file whatever you like. Just provided the full path to it when running fuzz-ab using the `--path=` flag.

The `ab-options` section can be comprised of any options specified in [AB's documentation](https://httpd.apache.org/docs/2.4/programs/ab.html) 

```
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
        "num_runs": 20,
        "num_per_run": 2,
        "concurrent": 5,
        "url-vars": {
            "some_slug": ["postslug1", "postslug2"]
        },
        "body": {
            "id": 5,
            "some_values": [1,2,3,4,5],
            "some_param": "{{some_param_value}}",
            "another_param": {
                "property1": 123,
                "property2": "{{property2_value}}"
            }
        },
        "body-vars": {
            "some_param_value": ["value1", "value2", "value3"],
            "property2_value": [1,2,3,4,5]
        },
        "ab-options": {
            "-n": 10,
            "-c": 5,
            "-C" : "session=123abc",
            "-H": "csrftoken: abc123"
        }
    }
}

```
