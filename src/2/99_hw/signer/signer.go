package main

import (
	"fmt"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// сюда писать код
func ExecutePipeline(jobs ...job) {
	prevOutput := make(chan interface{}, 100)
	wg := &sync.WaitGroup{}
	for idx, joba := range jobs {
		wg.Add(1)
		//input job
		output := make(chan interface{}, 100)
		fmt.Println("Running job with Index", idx)
		go jobWorker(joba, prevOutput, output, wg)
		prevOutput = output
	}
	wg.Wait()
}

func jobWorker(curJob job, in, out chan interface{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(out)
	curJob(in, out)
}

// SingleHash считает значение crc32(data)+"~"+crc32(md5(data))
// ( конкатенация двух строк через ~), где data - то что пришло на вход (по сути - числа из первой функции)
func SingleHash(in, out chan interface{}) {
	singleHashWg := &sync.WaitGroup{}
	fmt.Println("Started Singlehash")
	for inValue := range in {
		singleHashWg.Add(1)
		go func(inValue interface{}, out chan interface{}) {
			defer singleHashWg.Done()
			md5 := lockMD5Call(fmt.Sprintf("%#v", inValue))
			valueSlice := []string{fmt.Sprintf("%#v", inValue), md5}
			resultSlice := make([]string, 2)
			singleHashLeafWg := &sync.WaitGroup{}
			for idx, iter := range valueSlice {
				singleHashLeafWg.Add(1)
				go func(idx int, value string) {
					defer singleHashLeafWg.Done()
					r := DataSignerCrc32(value)
					resultSlice[idx] = r
					fmt.Println(inValue, "SingleHash result", r, "step", idx)
					runtime.Gosched()
				}(idx, iter)
			}
			singleHashLeafWg.Wait()
			result := strings.Join(resultSlice, "~")
			fmt.Println(inValue, "SingleHash result", result)
			out <- result
			runtime.Gosched()
		}(inValue, out)
	}
	singleHashWg.Wait()
}

// MultiHash считает значение crc32(th+data)) (конкатенация цифры, приведённой к строке и строки),
// где th=0..5 ( т.е. 6 хешей на каждое входящее значение ), потом берёт конкатенацию результатов в порядке расчета (0..5),
// где data - то что пришло на вход (и ушло на выход из SingleHash)
func MultiHash(in, out chan interface{}) {
	fmt.Println("Started MultiHash")
	multihashWg := &sync.WaitGroup{}
	for inValue := range in {
		multihashWg.Add(1)
		go func(inValue interface{}, out chan interface{}) {
			defer multihashWg.Done()
			resultSlice := make([]string, 6)
			multiHashLeafWg := &sync.WaitGroup{}
			for i := 0; i <= 5; i++ {
				i := i
				multiHashLeafWg.Add(1)
				go func(value interface{}) {
					defer multiHashLeafWg.Done()
					r := DataSignerCrc32(strconv.Itoa(i) + value.(string))
					resultSlice[i] = r
					fmt.Println(inValue, "MultiHash: crc32(th+step1))", i, r)
					runtime.Gosched()
				}(inValue)
			}
			multiHashLeafWg.Wait()
			result := strings.Join(resultSlice, "")
			fmt.Println("MultiHash for", inValue, "is", result)
			out <- result
			runtime.Gosched()
		}(inValue, out)
	}
	multihashWg.Wait()
}

// CombineResults получает все результаты, сортирует (https://golang.org/pkg/sort/),
// объединяет отсортированный результат через _ (символ подчеркивания) в одну строку
func CombineResults(in, out chan interface{}) {
	fmt.Println("Started Combine")
	var resultSlice []string
	for inValue := range in {
		resultSlice = append(resultSlice, inValue.(string))
	}
	sort.Strings(resultSlice)
	fmt.Println("poisonPill on CombineResults, returning result")
	result := strings.Join(resultSlice, "_")
	fmt.Println("Result", result)
	out <- result

}

var lockMDMutex = &sync.Mutex{}

func lockMD5Call(dataStr string) (md5 string) {
	lockMDMutex.Lock()
	defer lockMDMutex.Unlock()
	md5 = DataSignerMd5(dataStr)
	return
}
