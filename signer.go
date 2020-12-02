package main

import (
	"fmt"
)

func main() {
	testResult := "NOT_SET"
	//inputData := []int{0, 1, 1, 2, 3, 5, 8}
	inputData := []int{0,1}

	hashSignJobs := []job{
		job(func(in, out chan interface{}) {
			for _, fibNum := range inputData {
				out <- fibNum
			}
		}),
		job(SingleHash),
		job(MultiHash),
		job(CombineResults),
		job(func(in, out chan interface{}) {
			dataRaw := <-in
			data, ok := dataRaw.(string)
			if !ok {
				fmt.Println("cant convert result data to string")
			}
			testResult = data
		}),
	}

	//start := time.Now()

	ExecutePipeline(hashSignJobs...)

	//end := time.Since(start)
	fmt.Println(testResult)
/*
	//------------------------------
	in, out := make(chan interface{}), make(chan interface{})
	go func() {
		SingleHash(in, out)
		close(out)
	}()
	in <- 0
	singleHash1 := <- out
	in <- 1
	singleHash2 := <- out
	fmt.Println(singleHash1)
	fmt.Println(singleHash2)

	//------------------------------
	in, out = make(chan interface{}), make(chan interface{})
	go func() {
		MultiHash(in, out)
		close(out)
	}()
	in <- singleHash2
	multiHash1 := <- out
	in <- singleHash1
	multiHash2 := <- out
	fmt.Println(multiHash1)
	fmt.Println(multiHash2)

	//------------------------------
	in, out = make(chan interface{}), make(chan interface{})
	go func() {
		CombineResults(in, out)
		close(out)
	}()
	in <- multiHash1
	in <- multiHash2
	fmt.Println(<-out)
*/
}

// сюда писать код
func ExecutePipeline(jobs ...job) {
	// Объявляем переменные для входног/выходного каналов
	var in chan interface{}
	var out chan interface{}
	
	
	// Бежим по массиву задач и запускаем горутины
	for i, _ := range jobs {
		if i == 0 {
			in = make(chan interface{})
		} else {
			in <- out
		}

		if i < len(jobs) {
			out = make(chan interface{})
		} else {
			out = make(chan interface{})
		}

		go jobs[i](in, out)
	}
}

