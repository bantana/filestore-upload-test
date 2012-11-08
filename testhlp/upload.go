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

package testhlp

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"
)

// The Uploader interface provides upload/download functions
type Uploader interface {
	Upload(payload Payload) (string, error) // upload payload
	Get(string) (io.ReadCloser, error)      // get back data from url
}

// OneRound is the main function: runs one round of parallel uploads with concurrent reads
func OneRound(up Uploader, parallel, N int, urlch chan<- string, dump bool) (err error) {
	errch := make(chan error, 1+parallel)
	donech := make(chan uint64, parallel)
	upl := func(dump bool) {
		bp := uint64(0)
		var url string
		for i := 0; i < N; i++ {
			payload, err := getPayload("")
			if err != nil {
				errch <- fmt.Errorf("error getting payload(%d): %s", i, err)
				break
			}
			for j := rand.Int() % 15; j < 2; j++ {
				if url, err = CheckedUpload(up, payload, dump || bp < 1); err != nil {
					errch <- fmt.Errorf("error uploading: %s", err)
					break
				}
				bp += payload.Length
				select {
				case urlch <- url:
				default:
				}
			}
		}
		donech <- bp
	}
	for j := 0; j < parallel; j++ {
		go upl(dump && j < 1)
	}
	gbp := uint64(0)
	for i := 0; i < parallel; {
		select {
		case err = <-errch:
			log.Printf("ERROR: %s", err)
			return
		case b := <-donech:
			i++
			gbp += b
		}
	}
	log.Printf("done %d bytes", gbp)
	return nil
}

// uploads and checks (reads back data) right after the upload
func CheckedUpload(up Uploader, payload Payload, dump bool) (url string, err error) {
	if dump {
		log.Printf("Content-Type=%s", payload.ContentType)
	}
	hr, ok := payload.Data.(HashedReader)
	if !ok {
		hr = NewHashedReader(payload.Data)
		payload.Data = hr
	}
	url, err = up.Upload(payload)
	uphash := hr.Sum()
	if err != nil {
		return url, err
	}
	if url == "" {
		return url, fmt.Errorf("empty url!")
	}
	var r io.ReadCloser
	for i := 0; i < 10; i++ {
		if r, err = up.Get(url); err == nil {
			length, downhash, err := Hash(r)
			if err != nil {
				return url, err
			}
			if length != payload.Length {
				return url, fmt.Errorf("length mismatch for %s", url)
			}
			if !bytes.Equal(downhash, uphash) {
				return url, fmt.Errorf("hash mismatch for %s", url)
			}
			return url, nil
		}
		log.Printf("WARN[%d] cannot get %s: %s", i, url, err)
		time.Sleep(1)
	}
	return
}

var (
	hsh       hash.Hash
	hsh_mtx   = sync.Mutex{}
	NewHasher = sha256.New
)

// returns a hash of the data given by the reader
func Hash(r io.Reader) (uint64, []byte, error) {
	hsh_mtx.Lock()
	defer hsh_mtx.Unlock()
	if hsh == nil {
		hsh = NewHasher()
	} else {
		hsh.Reset()
	}
	length, err := io.Copy(hsh, r)
	if err != nil {
		return 0, nil, err
	}
	return uint64(length), hsh.Sum(nil), nil
}

type HashedReader interface {
	io.Reader
	Sum() []byte //returns the read data's hash value
}

type hashedReader struct {
	io.Reader
	hsh hash.Hash
}

func NewHashedReader(r io.Reader) HashedReader {
	if hr, ok := r.(HashedReader); ok {
		return hr
	}
	hsh := NewHasher()
	return &hashedReader{Reader: io.TeeReader(r, NewHasher()), hsh: hsh}
}

func (r *hashedReader) Sum() []byte {
	return r.hsh.Sum(nil)
}

func GetUrl(url string) (io.ReadCloser, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error with http.Get(%s): %s", url, err)
	}
	if resp == nil {
		return nil, fmt.Errorf("nil response for %s!", url)
	}
	if !(200 <= resp.StatusCode && resp.StatusCode <= 299) {
		return nil, fmt.Errorf("STATUS=%s (%s)", resp.Status, url)
	}
	return resp.Body, nil
}

func (payload Payload) Post(url string) (respBody []byte, err error) {
	var n int64
	reqbuf := bytes.NewBuffer(nil)
	n, err = EncodePayload(reqbuf, payload.Data, fmt.Sprintf("test-%d", payload.Length))
	if err != nil {
		return
	}
	req, e := http.NewRequest("POST", url, reqbuf)
	if e != nil {
		err = e
		return
	}
	req.ContentLength = n
	req.Header.Set("Content-Type", payload.ContentType)
	/*
	   if dump {
	       buf, e := httputil.DumpRequestOut(req, true)
	       if e != nil {
	           log.Panicf("cannot dump request %v: %s", req, e)
	       } else {
	           log.Printf("\n>>>>>>\nrequest:\n%v", buf)
	       }
	   }
	*/
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		buf, e := httputil.DumpRequestOut(req, true)
		if e != nil {
			log.Printf("cannot dump request %v: %s", req, e)
			return nil, err
		} else {
			log.Printf("\n>>>>>>\nrequest:\n%v", buf)
		}
	}
	if e != nil {
		err = e
		return
	}
	defer resp.Body.Close()
	if resp.ContentLength > 0 {
		respBody = make([]byte, resp.ContentLength)
		length, e := io.ReadFull(resp.Body, respBody)
		if e != nil {
			err = e
			return
		}
		respBody = respBody[:length]
	} else {
		respBody, e = ioutil.ReadAll(resp.Body)
	}

	if e != nil { //|| dump {
		err = e
		buf, e := httputil.DumpResponse(resp, true)
		if e != nil {
			log.Printf("cannot dump response %v: %s", resp, e)
		} else {
			log.Printf("\n<<<<<<\nresponse:\n%v", buf)
		}
		return nil, err
	}
	return
}
