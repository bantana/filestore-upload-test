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
	"net/http"
	"os"
	"runtime"
	"time"
)

// if called from command-line, start the server and push it under load!
func main() {
	aostorHp := flag.String("aostor", "", "aostor's server address host:port/realm")
	weedHp := flag.String("weed", "", "weed-fs master server address host:port")
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

	urlch := make(chan string, 1000)
	defer close(urlch)
	go func(urlch <-chan string) {
		for url := range urlch {
			resp, e := http.Get(url)
			if resp != nil && resp.Body != nil {
				resp.Body.Close()
				time.Sleep(1)
			}
			if e != nil {
				log.Printf("error with http.Get(%s): %s", url, e)
				os.Exit(1)
			}
			if !(200 <= resp.StatusCode && resp.StatusCode <= 299) {
				log.Printf("STATUS=%s for %s", resp.Status, url)
				os.Exit(2)
			}
		}
	}(urlch)
	// if *stage_interval > 0 {
	// 	ticker := time.Tick(time.Duration(*stage_interval) * time.Second)
	// 	// defer close(ticker)
	// 	go func(ch <-chan time.Time, hostport string) {
	// 		for now := range ch {
	// 			log.Printf("starting shovel at %s...", now)
	// 			if err = testhlp.Shovel(srv.Pid, hostport); err != nil {
	// 				log.Printf("error with shovel: %s", err)
	// 				break
	// 			}
	// 		}
	// 	}(ticker, *hostport)
	// }

	runtime.GOMAXPROCS(runtime.NumCPU())
	var err error
	for i := 2; i < 100; i++ {
		log.Printf("starting round %d...", i)
		if err = testhlp.OneRound(up, i, 100, urlch, i == 1); err != nil {
			log.Printf("error with round %d: %s", i, err)
			break
		}
	}
}
