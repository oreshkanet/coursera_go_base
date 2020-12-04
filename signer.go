package main

import (
	"fmt"
	"runtime"
	"time"
)

func main() {
	testResult := "NOT_SET"
	inputData := []int{0, 1, 1, 2, 3, 5, 8}
	//inputData := []int{0, 1}

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
			println("jobend " + testResult)
		}),
	}

	start := time.Now()
	ExecutePipeline(hashSignJobs...)
	end := time.Since(start)

	fmt.Println("Время выполненияЖ %v", end)
	fmt.Println(testResult)
}

// сюда писать код
func ExecutePipeline(jobs ...job) {
	// Объявляем переменные для входног/выходного каналов
	var in chan interface{}
	var out chan interface{}

	chans := make([]chan interface{}, 0)

	// Бежим по массиву задач и запускаем горутины
	for i, _ := range jobs {
		chans = append(chans, make(chan interface{}))

		if i == 0 {
			in = nil
		} else {
			in = chans[i-1]
		}
		out = chans[i]

		go func(job job, in chan interface{}, out chan interface{}) {
			job(in, out)
			runtime.Gosched()
			close(out)
		}(jobs[i], in, out)
	}
	for {
		if _, ok := <-out; !ok {
			break
		}
	}
	// Нужно получить значение из последнего потока
	// Иначе выполнение основной горутины кончится раньше, чем все остальные
}
