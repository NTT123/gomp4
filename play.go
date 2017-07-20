package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"
)

type DT struct {
	Data []byte
	At   int64
}

type Con struct {
	Min int64
	Max int64
}

var wg sync.WaitGroup

func main() {

	if len(os.Args) != 3 {
		fmt.Printf("Usage: %s filename url\n", os.Args[0])
		return
	}

	fmt.Println("Downloading!")

	url := os.Args[2]
	filename := os.Args[1]

	res, _ := http.Get(url)
	maps := res.Header
	length, _ := strconv.ParseInt(maps["Content-Length"][0], 10, 64)

	f, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE, 0655)
	loc, err := f.Seek(length, 0)

	if err != nil || loc != length {
		fmt.Println("SEEK ERR!")
		return
	}

	ch := make(chan DT, 100)
	chcon := make(chan Con)

	var limit int64 = 50
	len_sub := length / limit

	cons := make([]Con, limit)
	cons[0] = Con{0, len_sub}
	cons[1] = Con{length - len_sub, length}

	go func() {
		var i int64
		chcon <- cons[0]
		chcon <- cons[1]

		for i = 2; i < limit; i++ {
			cons[i] = Con{(i - 1) * len_sub, i * len_sub}
			chcon <- cons[i]
		}
	}()

	ncon := 6

	wg.Add(int(limit))

	go func() {
		cc := 0
		for c := range ch {
			f.Seek(c.At, 0)
			f.Write(c.Data)
			f.Sync()
			wg.Done()
			cc = cc + 1
			fmt.Printf("\r  %0.2f%%   ", float64(cc)*100.0/float64(limit))
		}
	}()

	var i int64

	for i = 0; i < int64(ncon); i++ {

		go func() {
			for co := range chcon {
				min := co.Min
				max := co.Max

				for {
					timeout := time.Duration(20 * time.Second)
					client := &http.Client{Timeout: timeout}
					req, _ := http.NewRequest("GET", url, nil)
					range_header := fmt.Sprintf("bytes=%d-%d", min, max-1)
					req.Header.Add("Range", range_header)
					resp, _ := client.Do(req)
					reader, _ := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					if int64(len(reader)) == max-min {
						ch <- DT{reader, min}
						break
					}

					fmt.Print("\r!")
				}

			}
		}()
	}

	wg.Wait()
	f.Close()
	fmt.Println()
}
