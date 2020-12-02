package main

import (
	"fmt"
	"sync/atomic"
	"time"
)

func main() {
	var ok = true
	var recieved uint32
	freeFlowJobs := []job{
		job(func(in, out chan interface{}) {
			out <- 1
			time.Sleep(10 * time.Millisecond)
			currRecieved := atomic.LoadUint32(&recieved)
			// в чем тут суть
			// если вы накапливаете значения, то пока вся функция не отрабоатет - дальше они не пойдут
			// тут я проверяю, что счетчик увеличился в следующей функции
			// это значит что туда дошло значение прежде чем текущая функция отработала
			if currRecieved == 0 {
				ok = false
			}
		}),
		job(func(in, out chan interface{}) {
			for _ = range in {
				atomic.AddUint32(&recieved, 1)
			}
		}),
	}
	ExecutePipeline(freeFlowJobs...)
	if !ok || recieved == 0 {
		fmt.Print("no value free flow - dont collect them")
	}
}

// сюда писать код
func ExecutePipeline(jobs ...job) {
	var in, out chan interface{}
	for i, v := range jobs {
		if i == 0 {
			in = make(chan interface{})
		} else {
			in = out
		}

		out = make(chan interface{})

		go v(in, out)
	}
}
