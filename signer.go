package main

import (
	"fmt"
	"runtime"
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
			println("jobend " + testResult)
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
			//runtime.Gosched()
			close(outGo)
		}(jobs[i], in, out)

		/* else {
			go func(jobs job, in chan interface{}, out chan interface{}) {
				wg := &sync.WaitGroup{}
				for data := range in {
					inJob := make(chan interface{}, 0)
					wg.Add(1)
					go func(wg *sync.WaitGroup, jobs job, in chan interface{}, out chan interface{}) {
						defer wg.Done()
						jobs(in, out)
						runtime.Gosched()
						//close(out)
					}(wg, jobs, inJob, out)
					inJob <- data
				}
				wg.Wait()
				close(out)
			}(jobs[i], in, out)
		}
		*/
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
		println("SingleHash <- " + fmt.Sprint(item))
	}

	goDataSignerCrc32 := func(dataIn string, outCrc chan string) {
		//defer wg.Done()
		outCrc <- DataSignerCrc32(dataIn)
	}

	startWG := time.Now()
	wg := &sync.WaitGroup{}
	for _, data := range data {
		//for data := range in {
		dataMd5 := DataSignerMd5(fmt.Sprint(data))
		inJob := make(chan interface{})
		wg.Add(1)
		go func(wg *sync.WaitGroup, inJob, outJob chan interface{}) {
			defer wg.Done()

			item := fmt.Sprint(<-inJob)
			start := time.Now()
			println("SingleHash " + item)
			//outJob <- DataSignerCrc32(item) + "~" + DataSignerCrc32(DataSignerMd5(item))
			//end := time.Since(start)
			//println("SingleHash " + fmt.Sprint(item) + " " + fmt.Sprint(end))

			//wgCrc := &sync.WaitGroup{}
			chSrc32Out := make(chan string, 0)
			chSrc32Out1 := make(chan string, 0)
			go goDataSignerCrc32(item, chSrc32Out)
			go goDataSignerCrc32(dataMd5, chSrc32Out1)

			end := time.Since(start)
			outJob <- (<-chSrc32Out + "~" + <-chSrc32Out1)

			println("SingleHash " + fmt.Sprint(item) + " " + fmt.Sprint(end))

		}(wg, inJob, out)
		inJob <- data
	}
	wg.Wait()
	endWG := time.Since(startWG)
	println("SingleHash " + fmt.Sprint(endWG))
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
			//th := []int{0, 1, 2, 3, 4, 5}
			resultSlice := make([]chan string, 5, 5)
			result := ""
			for i := range resultSlice {
				resultSlice[i] = make(chan string, 1)
				wg.Add(1)
				go func(wg *sync.WaitGroup, ch chan string, data string) {
					defer wg.Done()
					ch <- DataSignerCrc32(data)
					runtime.Gosched() // даём поработать другим горутинам
					//close(ch)
				}(wg, resultSlice[i], fmt.Sprint(i)+data)
				//result += DataSignerCrc32(fmt.Sprint(i) + data)
			}
			wg.Wait()
			for _, ch := range resultSlice {
				result += <-ch
			}
			//end := time.Since(start)
			//println("MultiHash " + result + " " + fmt.Sprint(end))

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
	for _, v := range data {
		result = result + "_" + v
	}

	//endWG := time.Since(startWG)
	//println("CombineResults result " + result + " " + fmt.Sprint(endWG))
	out <- result

}

// сюда писать код
func ExecutePipeline4(jobs ...job) {
	// Объявляем переменные для входног/выходного каналов
	//in := make(chan interface{}, 0)
	//out := make(chan interface{}, 0)

	//curI := 0

	var in chan interface{}
	var out chan interface{}

	chans := make([]chan interface{}, 0)

	// Бежим по массиву задач и запускаем горутины
	//for i, _ := range jobs {
	//for i: = 0; i < len(jobs); i++ {
	for i := 0; i < len(jobs); i++ {
		chans = append(chans, make(chan interface{}, 999))

		//out := make(chan interface{}, 0)

		if i == 0 {
			in = nil
		} else {
			in = chans[i-1]
		}
		if i == len(jobs)-1 {
			out = nil
		} else {
			out = chans[i]
		}

		go ExecuteGoroutine(jobs[i], in, out)
	}
	for {
		if _, ok := <-out; !ok {
			break
		}
	}

}

func ExecuteGoroutine(jobs job, in, out chan interface{}) {

	//inGo := make(chan interface{}, 9999)
	//outGo := make(chan interface{}, 9999)

	if in == nil {
		fmt.Printf("start\n")
		//dataIn <- in
		inJob := make(chan interface{}, 9999)
		outJob := make(chan interface{}, 9999)

		//ExecuteGoroutine(jobs, curI+1,
		go func(inJob, outJob chan interface{}) {
			jobs(inJob, outJob)
			close(outJob)
		}(inJob, outJob)
		//inJob <- dataIn
		runtime.Gosched()

		for dataOut := range outJob {
			fmt.Printf("dataOut %v\n", dataOut)
			out <- dataOut
		}

		close(out)
	} else if out == nil {
		go func(inJob, outJob chan interface{}) {
			jobs(inJob, outJob)
		}(in, out)
	} else {
		for dataIn := range in {
			fmt.Printf("dataIN %v\n", dataIn)
			//dataIn <- in
			inJob := make(chan interface{}, 9999)
			outJob := make(chan interface{}, 9999)

			//ExecuteGoroutine(jobs, curI+1,
			go func(inJob, outJob chan interface{}) {
				jobs(inJob, outJob)
				close(outJob)
			}(inJob, outJob)
			inJob <- dataIn
			runtime.Gosched()

			for dataOut := range outJob {
				fmt.Printf("dataOut %v\n", dataOut)
				out <- dataOut
			}
		}

		close(out)
	}

	/*

			for dataOut := range outJob {
				fmt.Printf("curI %v, dataOut %v\n", curI, dataOut)
				if curI == len(jobs)-1 {
					out <- dataOut
				} else {
					inGo := make(chan interface{}, 9999)
					outGo := make(chan interface{}, 9999)
					go ExecuteGoroutine(jobs, curI+1, inGo, outGo)
					inGo <- dataOut
					runtime.Gosched()
					for dataGo := range outGo {
						fmt.Printf("curI %v, dataGo %v\n", curI, dataGo)
						out <- dataGo
						//
					}
					close(outGo)

				}
			}

		}
	*/

}

func ExecuteGoroutine1(jobs []job, curI int, in chan interface{}) {

	//inGo := make(chan interface{}, 9999)
	//outGo := make(chan interface{}, 9999)
	for dataIn := range in {
		fmt.Printf("curI %v, dataIN %v\n", curI, dataIn)
		//dataIn <- in
		inJob := make(chan interface{}, 0)
		outJob := make(chan interface{}, 0)

		//ExecuteGoroutine(jobs, curI+1,
		go jobs[curI](inJob, outJob)
		inJob <- dataIn

	}
	//go ExecuteGoroutine(jobs, curI+1, out)

	/*

			for dataOut := range outJob {
				fmt.Printf("curI %v, dataOut %v\n", curI, dataOut)
				if curI == len(jobs)-1 {
					out <- dataOut
				} else {
					inGo := make(chan interface{}, 9999)
					outGo := make(chan interface{}, 9999)
					go ExecuteGoroutine(jobs, curI+1, inGo, outGo)
					inGo <- dataOut
					runtime.Gosched()
					for dataGo := range outGo {
						fmt.Printf("curI %v, dataGo %v\n", curI, dataGo)
						out <- dataGo
						//
					}
					close(outGo)

				}
			}

		}
	*/

}

func ExecutePipeline2(jobs ...job) {
	// Объявляем переменные для входног/выходного каналов
	var in chan interface{}
	var out chan interface{}

	chans := make([]chan interface{}, 0)

	// Бежим по массиву задач и запускаем горутины
	for i, _ := range jobs {
		chans = append(chans, make(chan interface{}, 999))

		if i == 0 {
			in = nil
		} else {
			in = chans[i-1]
		}
		out = chans[i]

		/*
			go func(job job, in chan interface{}, out chan interface{}) {
				job(in, out)
				runtime.Gosched()
				close(out)
			}(jobs[i], in, out)
		*/

		//go func(job job, in chan interface{}, out chan interface{}) {
		if i == 0 {
			//inJob := make(chan interface{})
			outJob := make(chan interface{}, 10)
			go jobs[i](nil, outJob)
			//inJob <- data
			//runtime.Gosched()
			//out <- <-outJob
			for data := range outJob {
				out <- data
			}
		} else {
			for data := range in {
				inJob := make(chan interface{})
				outJob := make(chan interface{})
				go jobs[i](inJob, outJob)
				inJob <- data
				//runtime.Gosched()
				out <- <-outJob
				//runtime.Gosched()
				close(outJob)
			}

		}
		//close(out)
		//}(jobs[i], in, out)
	}
	for {
		if _, ok := <-out; !ok {
			break
		}
	}
	// Нужно получить значение из последнего потока
	// Иначе выполнение основной горутины кончится раньше, чем все остальные
}
