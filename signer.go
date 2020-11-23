package main

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

const TH = 6

func ExecutePipeline(jobs ...job){
	wg := &sync.WaitGroup{}
	in := make(chan interface{})

	for _, jobFunc := range jobs{
		wg.Add(1)
		out := make(chan interface{})
		go workerPipeline(wg, jobFunc, in, out)
		in = out
	}
	wg.Wait()

}

func workerPipeline(wg *sync.WaitGroup, jobFunc job, in, out chan interface{}){
	defer wg.Done()
	defer close(out)
	jobFunc(in, out)
}

func SingleHash(in, out chan interface{}){
	wg := &sync.WaitGroup{}
	for i := range in {
		data := fmt.Sprintf("%v", i)
		crcMd5 := DataSignerMd5(data)
		wg.Add(1)
		go workerSingleHash(wg, data, crcMd5, out)
	}
	wg.Wait()
}

func workerSingleHash(wg *sync.WaitGroup, data string, crcMd5 string, out chan interface{}){
	defer wg.Done()

	crc32Chan := make(chan string)
	crcMd5Chan := make(chan string)

	go calculateHash(crc32Chan, data, DataSignerCrc32)
	go calculateHash(crcMd5Chan, crcMd5, DataSignerCrc32)

	crc32Hash := <-crc32Chan
	crc32Md5Hash := <-crcMd5Chan

	out <- crc32Hash + "~" + crc32Md5Hash

}

func calculateHash(ch chan string, data string, f func(string) string){
	res := f(data)
	ch <- res
}

func MultiHash(in, out chan interface{}){
	wg := &sync.WaitGroup{}
	for i := range in {
		wg.Add(1)

		go workerMultiHash(wg, i, out)
	}

	wg.Wait()
}

func workerMultiHash(wg *sync.WaitGroup, h interface{}, ch chan interface{}){

	wgInternal :=  &sync.WaitGroup{}
	hashArray := make([]string, TH)

	defer wg.Done()

	for idx := 0; idx < TH; idx++ {
		wgInternal.Add(1)
		data := fmt.Sprintf("%v%v", idx, h)
		go calculateMultiHash(wgInternal, data, hashArray, idx)
	}
	wgInternal.Wait()
	multiHash := strings.Join(hashArray, "")

	ch <- multiHash
}

func calculateMultiHash(wg *sync.WaitGroup, s string, array []string, index int){
	defer wg.Done()
	crc32hash := DataSignerCrc32(s)
	array[index] = crc32hash
}

func CombineResults(in, out chan interface{}){
	var hashArray []string

	for i := range in {
		hashArray = append(hashArray, i.(string))
	}

	sort.Strings(hashArray)
	combineResults := strings.Join(hashArray, "_")
	out <- combineResults
}