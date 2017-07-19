package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"sync"
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
	fmt.Println("http://127.0.0.1:8888/")
	f, _ := os.OpenFile("hi.mp4", os.O_TRUNC|os.O_WRONLY|os.O_CREATE, 0755)

	res, _ := http.Get(os.Args[1])
	maps := res.Header
	length, _ := strconv.ParseInt(maps["Content-Length"][0], 10, 64)

	ch := make(chan DT, 100)
	chcon := make(chan Con)

	var limit int64 = 100
	len_sub := length / limit
	//diff := length % limit
	cons := make([]Con, limit)
	cons[0] = Con{0, len_sub}
	cons[1] = Con{length - 1024*1024, length}
	cons[2] = Con{length - 1024*1024, length}
	cons[3] = Con{length - 1024*1024, length}
	cons[4] = Con{length - 1024*1024, length}

	go func() {
		var i int64
		chcon <- cons[0]
		chcon <- cons[1]
		chcon <- cons[2]
		chcon <- cons[3]
		chcon <- cons[4]

		for i = 2; i < limit; i++ {
			fmt.Printf("\r  %0.2f%%   ", float64(i)*100.0/float64(limit))
			cons[i] = Con{(i - 1) * len_sub, i * len_sub}
			chcon <- cons[i]
		}
	}()

	ncon := 6

	wg.Add(int(limit))

	go func() {

		all := make([]byte, length)

		go func() {
			handler := func(w http.ResponseWriter, r *http.Request) {
				if _, ok := r.Header["Range"]; ok {
					s := r.Header["Range"][0][6:]
					var min, max int64
					fmt.Scanf(s, "%d-%d", &min, &max)
					fmt.Println(min, max)

					client := &http.Client{}
					req, _ := http.NewRequest("GET", os.Args[1], nil)
					range_header := fmt.Sprintf("bytes=%d-%d", min, max)
					req.Header.Add("Range", range_header)
					resp, _ := client.Do(req)
					reader, _ := ioutil.ReadAll(resp.Body)
					resp.Body.Close()
					w.Write(reader)

				} else {
					for k, v := range maps {
						w.Header().Set(k, v[0])
					}

					io.WriteString(w, "Hello")
				}
			}

			http.HandleFunc("/", handler)
			http.ListenAndServe(":8888", nil)
		}()

		for c := range ch {
			copy(all[c.At:c.At+int64(len(c.Data))], c.Data)
			wg.Done()
		}
	}()

	var i int64
	for i = 0; i < int64(ncon); i++ {

		go func() {
			for co := range chcon {
				min := co.Min
				max := co.Max

				client := &http.Client{}
				req, _ := http.NewRequest("GET", os.Args[1], nil)
				range_header := fmt.Sprintf("bytes=%d-%d", min, max-1)
				req.Header.Add("Range", range_header)
				resp, _ := client.Do(req)
				reader, _ := ioutil.ReadAll(resp.Body)
				resp.Body.Close()
				ch <- DT{reader, min}

			}
		}()
	}
	wg.Wait()
	f.Close()

}
