package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

// сюда писать код

var su = sync.Mutex{}

func ExecutePipeline(jobs ...job) {
	wg := sync.WaitGroup{}
	in := make(chan interface{}, 1)
	for _, jobf := range jobs {
		wg.Add(1)
		out := make(chan interface{}, 1)
		go func(job job, in, out chan interface{}, wg *sync.WaitGroup) {
			job(in, out)
			close(out)
			wg.Done()
		}(jobf, in, out, &wg)
		in = out
	}
	wg.Wait()
}

func SingleHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	for val := range in {
		wg.Add(1)
		go func(val interface{}, wg *sync.WaitGroup) {
			str := fmt.Sprint(val)
			firstRes := make(chan string, 1)
			go getCrc32(str, firstRes)
			secondRes := make(chan string, 1)
			go getMd5(str, secondRes)
			go getCrc32(<-secondRes, secondRes)
			out <- <-firstRes + "~" + <-secondRes
			close(firstRes)
			close(secondRes)
			wg.Done()
		}(val, &wg)
	}
	wg.Wait()
}

func MultiHash(in, out chan interface{}) {
	wg := sync.WaitGroup{}
	for val := range in {
		wg.Add(1)
		go func(val interface{}, wg *sync.WaitGroup) {
			res := make(chan orderData, 6)
			for th := 0; th <= 5; th++ {
				go getCrc32WithOrder(fmt.Sprintf("%d%s", th, val), th, res)
			}
			result := [6]string{}
			for th := 0; th <= 5; th++ {
				data := <-res
				result[data.id] = data.data
			}
			out <- strings.Join(result[:], "")
			wg.Done()
		}(val, &wg)
	}
	wg.Wait()
}

func CombineResults(in, out chan interface{}) {

	result := make([]string, 0, 8)
	for val := range in {
		result = append(result, val.(string))
	}
	sort.Strings(result)
	out <- strings.Join(result, "_")
}

func getCrc32WithOrder(data string, id int, res chan orderData) {
	res <- orderData{id, DataSignerCrc32(data)}
}

func getCrc32(data string, res chan string) {
	res <- DataSignerCrc32(data)
}

func getMd5(data string, res chan string) {
	su.Lock()
	res <- DataSignerMd5(data)
	su.Unlock()
}

type orderData struct {
	id   int
	data string
}

func main() {
	ExecutePipeline(
		job(func(in, out chan interface{}) {
			for ind := range [1]string{} {
				out <- fmt.Sprintf("added - %d\n", ind)
			}
		}),
		SingleHash,
		job(func(in, out chan interface{}) {
			for val := range in {
				fmt.Printf("out - %d\n", val)
			}
		}),
	)
}
