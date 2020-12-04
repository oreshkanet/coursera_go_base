package main

import (
	"crypto/md5"
	"fmt"
	"hash/crc32"
	"strconv"
	"sync/atomic"
	"time"
)

type job func(in, out chan interface{})

const (
	MaxInputDataLen = 100
)

var (
	dataSignerOverheat uint32 = 0
	DataSignerSalt            = ""
)

var OverheatLock = func() {
	for {
		if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 0, 1); !swapped {
			fmt.Println("OverheatLock happend")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

var OverheatUnlock = func() {
	for {
		if swapped := atomic.CompareAndSwapUint32(&dataSignerOverheat, 1, 0); !swapped {
			fmt.Println("OverheatUnlock happend")
			time.Sleep(time.Second)
		} else {
			break
		}
	}
}

var DataSignerMd5 = func(data string) string {
	OverheatLock()
	defer OverheatUnlock()
	data += DataSignerSalt
	dataHash := fmt.Sprintf("%x", md5.Sum([]byte(data)))
	time.Sleep(10 * time.Millisecond)
	return dataHash
}

var DataSignerCrc32 = func(data string) string {
	data += DataSignerSalt
	crcH := crc32.ChecksumIEEE([]byte(data))
	dataHash := strconv.FormatUint(uint64(crcH), 10)
	time.Sleep(time.Second)
	return dataHash
}

var SingleHash = func(in, out chan interface{}) {
	for {
		if data, ok := <-in; ok {
			println("SingleHash " + fmt.Sprint(data))
			var item string
			var ok bool
			if item, ok = data.(string); !ok {
				item = fmt.Sprint(data)
			}
			//out <- DataSignerCrc32(item) + "~" + DataSignerCrc32(DataSignerMd5(item))
			chSrc32In := make(chan string)
			chSrc32Out := make(chan string)
			chSrc32In1 := make(chan string)
			chSrc32Out1 := make(chan string)
			chSrc32In2 := make(chan string)
			chSrc32Out2 := make(chan string)
			go func(in chan string, out chan string) {
				out <- DataSignerCrc32(<-in)
			}(chSrc32In, chSrc32Out)
			go func(in chan string, out chan string) {
				out <- DataSignerMd5(<-in)
			}(chSrc32In1, chSrc32Out1)
			go func(in chan string, out chan string) {
				out <- DataSignerCrc32(<-in)
			}(chSrc32In2, chSrc32Out2)
			chSrc32In <- item
			chSrc32In1 <- item
			chSrc32In2 <- chSrc32Out1
			out <- (<-chSrc32Out + "~" + <-chSrc32In2)
		} else {
			break
		}

	}
}

var MultiHash = func(in, out chan interface{}) {
	for {
		if item, ok := <-in; ok {
			data := fmt.Sprint(item)
			th := []int{0, 1, 2, 3, 4, 5}
			result := ""
			for _, v := range th {
				result += DataSignerCrc32(fmt.Sprint(v) + data)
			}
			println("MultiHash " + data)
			out <- result
		} else {
			break
		}

	}
}

var CombineResults = func(in, out chan interface{}) {
	result := ""
	data := make([]string, 0)
	for {
		if item, ok := <-in; ok {
			data = append(data, fmt.Sprint(item))
		} else {
			for _, v := range data {
				result = result + "_" + v
			}

			out <- result
			break
		}
	}
}
