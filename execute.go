package main

import (
	"fmt"
	"os/exec"
	"reflect"
)


func handle_request_group(request_data AbRequest, ch chan string) {
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


func Execute(requests []AbRequest) {
	var chans []chan string
	var total int
	for _, request := range requests {
		c := make(chan string)
		chans = append(chans, c)
		total += len(request.Requests)
		go handle_request_group(request, c)
	}

	fmt.Printf("%v ab calls to make\n", total)

	cases := make([]reflect.SelectCase, len(chans))
	for i, ch := range chans {
		cases[i] = reflect.SelectCase{Dir: reflect.SelectRecv, Chan: reflect.ValueOf(ch)}
	}

	remaining := len(cases)
	for remaining > 0 {
		chosen, value, ok := reflect.Select(cases)
		if !ok {
			cases[chosen].Chan = reflect.ValueOf(nil)
			remaining -= 1
			continue
		}

		fmt.Println(value.String())
	}
}