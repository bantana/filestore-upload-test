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
	"io"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"sync"
	"time"
)

var pushback bool = true

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
	flag.IntVar(&testhlp.SameOdds, "request.same", 0, "push same requests 1 out of N")
	flag.BoolVar(&testhlp.Compressable, "request.compressable", false, "should the request be compressable?")

	flag.Parse()

	if parallelWrite > 1 {
		requestNum = (requestNum + (parallelWrite + 1)) / parallelWrite
	}

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

	var (
		wg    *sync.WaitGroup
		urlch chan string
	)
	if parallelRead > 0 {
		urlch = make(chan string, 10000)
		wg = new(sync.WaitGroup)

		for i := 0; i < parallelRead; i++ {
			go reader(urlch, wg)
		}
	}

	runtime.GOMAXPROCS(runtime.NumCPU())
	var err error
	if err = testhlp.OneRound(up, parallelWrite, requestNum, urlch, true); err != nil {
		log.Printf("error: %s", err)
		os.Exit(9)
	}

	if parallelRead > 0 {
		pushback = false
		wg.Wait()
		close(urlch)
	}
	log.Printf("OK")
}

func reader(urlch chan string, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	var url string
	for {
		select {
		case url = <-urlch:
		default:
			if pushback {
				time.Sleep(1 * time.Second)
				continue
			} else {
				return
			}
		}
		log.Printf("GET " + url)
		body, e := testhlp.GetUrl(url)
		if e != nil {
			log.Printf("error with Get(%s): %s", url, e)
			os.Exit(1)
		}
		_, e = io.Copy(ioutil.Discard, body)
		if body != nil {
			body.Close()
		}
		if e != nil {
			log.Printf("error reading %s: %s", url, e)
			os.Exit(1)
		}
		// time.Sleep(50 * time.Millisecond)
		if pushback {
			select {
			case urlch <- url:
				// time.Sleep(10 * time.Millisecond)
				runtime.Gosched()
			default:
			}
		}
	}
}
