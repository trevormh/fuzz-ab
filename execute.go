package main

import (
	"fmt"
	"os/exec"
	"reflect"
	"regexp"
)

type Summary struct {
	Complete int
	Failed int
	PerSecond int
}

type Result struct {
	RawResult []string
	ExtractedResult []map[string]int
	Summary Summary
}


func (result *Result) extractSummaryFields(output string) {
	extracted := make(map[string]int)
	var re *regexp.Regexp

	re = regexp.MustCompile(`Complete requests:.*`)
	completed_line := re.FindString(output)
	re = regexp.MustCompile(`\d+`)
	extracted["complete"] = ConvertToInt(re.FindString(completed_line))

	re = regexp.MustCompile(`Failed requests:.*`)
	failed_line := re.FindString(output)
	re = regexp.MustCompile(`\d+`)
	extracted["failed"] = ConvertToInt(re.FindString(failed_line))

	re = regexp.MustCompile(`Requests per second:.*`)
	per_second_line := re.FindString(output)
	re = regexp.MustCompile(`\d+`)
	extracted["per_second"] = ConvertToInt(re.FindString(per_second_line))

	result.ExtractedResult = append(result.ExtractedResult, extracted)
}


func (results *Result) processResults() {
	complete := 0
	failed := 0
	per_second := 0
	for i, res := range results.RawResult {
		results.extractSummaryFields(res)
		complete += results.ExtractedResult[i]["complete"]
		failed += results.ExtractedResult[i]["failed"]
		per_second += results.ExtractedResult[i]["per_second"]
	}
	per_second = per_second / len(results.RawResult)
	
	results.Summary.Complete = complete
	results.Summary.Failed = failed
	results.Summary.PerSecond = per_second
}

func execute(request_data AbRequest, ch chan string) {
	for _, request := range request_data.Requests {
		cmd := exec.Command("ab", request...)
		stdout, err := cmd.Output()
		if err != nil {
			fmt.Println(err)
		}
		ch <- string(stdout)
	}
	close(ch)
}


func ExecuteRequests(requests []AbRequest, verbose bool) Result {
	var chans []chan string
	var total int
	for _, request := range requests {
		c := make(chan string)
		chans = append(chans, c)
		total += len(request.Requests)
		go execute(request, c)
	}

	fmt.Printf("%v ab calls to make\n", total)

	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	var result Result
	remaining := len(cases)
	for remaining > 0 {
		chosen, output, ok := reflect.Select(cases)
		if !ok {
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining -= 1
			continue
		}
		if verbose == true {
			fmt.Println(output.String())
		}
		result.RawResult = append(result.RawResult, output.String())
	}
	
	result.processResults()

	return result
}