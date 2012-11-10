// Copyright 2012 Tamás Gulácsi, UNO-SOFT Computing Ltd.

// file-upload-test is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// file-upload-test is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with file-upload-test.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"flag"
	"github.com/tgulacsi/filestore-upload-test/testhlp"
	"log"
	"math/rand"
	"os"
	"runtime"
	"sync"
	// "time"
)

// if called from command-line, start the server and push it under load!
func main() {
	var parallelRead, parallelWrite, requestNum int
	aostorHp := flag.String("aostor", "", "aostor's server address host:port/realm")
	weedHp := flag.String("weed", "", "weed-fs master server address host:port")
	flag.BoolVar(&testhlp.Dump, "dump", false, "dump?")
	flag.BoolVar(&testhlp.Debug, "debug", false, "debug?")
	flag.IntVar(&parallelRead, "parallel.read", 1, "read parallelism")
	flag.IntVar(&parallelWrite, "parallel.write", 1, "write parallelism")
	flag.IntVar(&requestNum, "request.num", 100, "request number")
	flag.BoolVar(&testhlp.GzipOk, "request.gzip", false, "request compressed?")
	flag.IntVar(&testhlp.PayloadSizeInit, "request.size.init", 1<<15, "request initial size, in bytes")
	flag.IntVar(&testhlp.PayloadSizeMax, "request.size.max", 1<<20, "request maximal size, in bytes")
	flag.IntVar(&testhlp.PayloadSizeStep, "request.size.step", 1<<15, "request size step, in bytes")

	flag.Parse()
	var up testhlp.Uploader
	switch {
	case aostorHp != nil && *aostorHp != "":
		if (*aostorHp)[:1] == ":" {
			*aostorHp = "localhost" + *aostorHp
		}
		up = &testhlp.Aostor{"http://" + *aostorHp}
	case weedHp != nil && *weedHp != "":
		if (*weedHp)[:1] == ":" {
			*weedHp = "localhost" + *weedHp
		}
		up = &testhlp.Weed{"http://" + *weedHp}
	default:
		log.Printf("http is required!")
		os.Exit(1)
	}
	// srv, err := testhlp.StartServer(*hostport)
	// if err != nil {
	// 	log.Panicf("error starting server: %s", err)
	// }
	// defer func() {
	// 	if srv.Close != nil {
	// 		srv.Close()
	// 	}
	// }()

	urlch := make(chan string, 10000)
	defer close(urlch)
	wg := sync.WaitGroup{}

	for i := 0; i < parallelRead; i++ {
		go reader(urlch, wg)
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	var err error
	for i := 1; i < parallelWrite+1; i++ {
		log.Printf("starting round %d...", i)
		if err = testhlp.OneRound(up, i, requestNum, urlch, i == 1); err != nil {
			log.Printf("error with round %d: %s", i, err)
			break
		}
	}

	for i := 0; i < parallelRead; i++ {
		urlch <- ""
	}
	wg.Wait()
}

func reader(urlch chan string, wg sync.WaitGroup) {
	wg.Add(1)
	for url := range urlch {
		if url == "" {
			wg.Done()
			return
		}
		log.Printf("GET " + url)
		body, e := testhlp.GetUrl(url)
		if body != nil {
			body.Close()
		}
		if e != nil {
			log.Printf("error with Get(%s): %s", url, e)
			os.Exit(1)
		}
		// time.Sleep(50 * time.Millisecond)
		if rand.Int()%5 != 0 {
			select {
			case urlch <- url:
			default:
			}
		}
	}
}
