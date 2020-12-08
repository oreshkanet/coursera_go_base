package main

import (
	"fmt"
	"runtime"
	"sort"
	"sync"
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
			// println("jobend " + testResult)
		}),
	}

	start := time.Now()
	ExecutePipeline(hashSignJobs...)
	end := time.Since(start)

	fmt.Printf("Время выполнения: %v\n", end)
	fmt.Println(testResult)
}

func ExecutePipeline(jobs ...job) {
	// Объявляем переменные для входног/выходного каналов
	var in chan interface{}
	var out chan interface{}

	chans := make([]chan interface{}, 0)

	// Бежим по массиву задач и запускаем горутины
	for i, _ := range jobs {
		chans = append(chans, make(chan interface{}, 100))

		if i == 0 {
			in = nil
		} else {
			in = chans[i-1]
		}
		out = chans[i]

		go func(job job, inGo chan interface{}, outGo chan interface{}) {
			job(inGo, outGo)
			close(outGo)
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

var SingleHash = func(in, out chan interface{}) {

	data := make([]string, 0)
	for item := range in {
		data = append(data, fmt.Sprint(item))
		//println("SingleHash <- " + fmt.Sprint(item))
	}

	goDataSignerCrc32 := func(dataIn string, outCrc chan string) {
		outCrc <- DataSignerCrc32(dataIn)
	}

	//startWG := time.Now()
	wg := &sync.WaitGroup{}
	for _, data := range data {
		dataMd5 := DataSignerMd5(fmt.Sprint(data))
		inJob := make(chan interface{})
		wg.Add(1)
		go func(wg *sync.WaitGroup, inJob, outJob chan interface{}) {
			defer wg.Done()

			item := fmt.Sprint(<-inJob)
			//start := time.Now()
			//println("SingleHash " + item)
			chSrc32Out := make(chan string, 0)
			chSrc32Out1 := make(chan string, 0)
			go goDataSignerCrc32(item, chSrc32Out)
			go goDataSignerCrc32(dataMd5, chSrc32Out1)

			outJob <- (<-chSrc32Out + "~" + <-chSrc32Out1)

			//end := time.Since(start)
			//println("SingleHash " + fmt.Sprint(item) + " " + fmt.Sprint(end))

		}(wg, inJob, out)
		inJob <- data
	}
	wg.Wait()
	//endWG := time.Since(startWG)
	//println("SingleHash " + fmt.Sprint(endWG))
}

var MultiHash = func(in, out chan interface{}) {
	data := make([]string, 0)
	for item := range in {
		data = append(data, fmt.Sprint(item))
		//println("MultiHash " + fmt.Sprint(item))
	}
	wgj := &sync.WaitGroup{}
	for _, data := range data {
		inJob := make(chan interface{})
		wgj.Add(1)
		go func(wggo *sync.WaitGroup, inJob, outJob chan interface{}) {
			defer wggo.Done()
			//start := time.Now()
			data := fmt.Sprint(<-inJob)
			runtime.Gosched()
			wg := &sync.WaitGroup{}
			resultSlice := make([]chan string, 6, 6)
			result := ""
			for i := range resultSlice {
				resultSlice[i] = make(chan string, 1)
				wg.Add(1)
				go func(wg *sync.WaitGroup, ch chan string, data string) {
					defer wg.Done()
					ch <- DataSignerCrc32(data)
					runtime.Gosched() // даём поработать другим горутинам
				}(wg, resultSlice[i], fmt.Sprint(i)+data)
			}
			wg.Wait()
			for _, ch := range resultSlice {
				result += <-ch
			}

			outJob <- result
		}(wgj, inJob, out)
		inJob <- data
	}
	wgj.Wait()
}

var CombineResults = func(in, out chan interface{}) {
	//startWG := time.Now()
	result := ""
	data := make([]string, 0)
	for item := range in {
		data = append(data, fmt.Sprint(item))
		//println("CombineResults " + fmt.Sprint(item))
	}

	sort.Slice(data, func(i, j int) bool { return data[i] < data[j] })
	for _, v := range data {
		if result != "" {
			result += "_"
		}
		result = result + v
	}

	//endWG := time.Since(startWG)
	//println("CombineResults result " + result + " " + fmt.Sprint(endWG))
	out <- result
}
