package main

import (
	"fmt"
	"sync"
	"time"
)

var or = func(channels ...<-chan interface{}) <-chan interface{} {
	res := make(chan interface{})
	var once sync.Once
	for i, channel := range channels {
		go func(ch <-chan interface{}, idx int) {
			select {
			case _, ok := <-channel:
				if !ok {
					once.Do(func() {
						fmt.Printf("res closed on channel with index = %v\n", idx)
						close(res)
					})
					return
				}
			}
		}(channel, i)
	}
	return res
}

func main() {
	sig := func(after time.Duration) <-chan interface{} {
		c := make(chan interface{})
		go func() {
			defer close(c)
			time.Sleep(after)
		}()
		return c
	}

	start := time.Now()
	<-or(
		sig(2*time.Hour),
		sig(5*time.Minute),
		sig(1*time.Second),
		sig(1*time.Hour),
		sig(1*time.Minute),
	)
	fmt.Printf("done after %v", time.Since(start))
}
