package main

import (
	"fmt"
	"math/rand"
	"os/exec"
	"reflect"
	"regexp"
	"time"
)

type Summary struct {
	Complete int
	Failed int
	PerSecond float32
}

type Result struct {
	RawResult []string
	ExtractedResult []Summary
	Summary Summary
}


func (result *Result) extractSummaryFields(output string) {
	var extracted Summary
	var re *regexp.Regexp

	re = regexp.MustCompile(`Complete requests:.*`)
	completed_line := re.FindString(output)
	re = regexp.MustCompile(`\d+`)
	extracted.Complete = ConvertToInt(re.FindString(completed_line))

	re = regexp.MustCompile(`Failed requests:.*`)
	failed_line := re.FindString(output)
	re = regexp.MustCompile(`\d+`)
	extracted.Failed = ConvertToInt(re.FindString(failed_line))

	re = regexp.MustCompile(`Requests per second:.*`)
	per_second_line := re.FindString(output)
	re = regexp.MustCompile(`\d+`)
	extracted.PerSecond = ConvertToFloat32(re.FindString(per_second_line))

	result.ExtractedResult = append(result.ExtractedResult, extracted)
}


func (results *Result) processResults() {
	complete := 0
	failed := 0
	var per_second float32
	per_second = 0.0
	for i, res := range results.RawResult {
		results.extractSummaryFields(res)
		complete += results.ExtractedResult[i].Complete
		failed += results.ExtractedResult[i].Failed
		per_second += results.ExtractedResult[i].PerSecond
	}
	per_second = per_second / float32(len(results.RawResult))
	
	results.Summary.Complete = complete
	results.Summary.Failed = failed
	results.Summary.PerSecond = per_second
}

func execute_request(request []string, ch chan string) {
	cmd := exec.Command("ab", request...)
	stdout, err := cmd.Output()

	if err != nil {
		fmt.Println(err.Error())
	}
	ch <- string(stdout)
	defer close(ch)
}

func execute_group(request AbRequest, ch chan []string) {
	var chans []chan string
	for _, req := range request.Requests {
		c := make(chan string)
		chans = append(chans, c)
		go execute_request(req, c)

		min, max := request.Delay[0], request.Delay[1]
		dur := min + rand.Intn(max - min + 1)
		if dur > 0 {
			time.Sleep(time.Duration(dur) * time.Millisecond)
		}
	}

	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	responses := make([]string, len(request.Requests))
	remaining := len(cases)

	for remaining > 0 {
		chosen, output, ok := reflect.Select(cases)
		if !ok {
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining -= 1
			continue
		}
		responses[chosen] = output.String()
	}
	ch <- responses
	close(ch)
}

func ExecuteRequests(requests []AbRequest, verbose bool) Result {
	var chans []chan []string
	var total int
	for _, group := range requests {
		c := make(chan []string)
		chans = append(chans, c)
		total += len(group.Requests)
		go execute_group(group, c)
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

		output_slice, ok := output.Interface().([]string)
		if !ok {
			panic("value not a []string")
		}
		for _, out := range output_slice {
			result.RawResult = append(result.RawResult, out)
		}
	}	
	result.processResults()

	return result
}