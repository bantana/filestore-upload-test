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
	"errors"
	"fmt"
	"hash"
	"io"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/http/httputil"
	"strings"
	"sync"
	"time"
)

var (
	Dump     bool = false
	GzipOk   bool = true
	hashIsOk bool = false
)

// The Uploader interface provides upload/download functions
type Uploader interface {
	Upload(payload Payload) (string, error) // upload payload
	Get(string) (io.ReadCloser, error)      // get back data from url
}

// OneRound is the main function: runs one round of parallel uploads with concurrent reads
func OneRound(up Uploader, parallel, N int, urlch chan<- string, dump bool) (err error) {

	parallel = 1

	if parallel <= 1 {
		log.Printf("calling uploadRound")
		err = uploadRound(up, N, urlch, nil, nil, dump)
		log.Printf("uploadRound: %s", err)
		return err
	}
	return

	errch := make(chan error, 1+parallel)
	donech := make(chan uint64, parallel)
	for j := 0; j < parallel; j++ {
		go uploadRound(up, N, urlch, donech, errch, dump && j < 1)
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

func uploadRound(up Uploader, N int, urlch chan<- string, donech chan<- uint64, errch chan<- error, dump bool) error {
	bp := uint64(0)
	defer func() {
		if donech != nil {
			select {
			case donech <- bp:
			default:
			}
		}
	}()
	var url string
	for i := 0; i < N; i++ {
		log.Printf(" i=%d < %d=N", i, N)
		payload, err := getPayload("")
		if err != nil {
			err = fmt.Errorf("error getting payload(%d): %s", i, err)
			log.Printf("err=%s", err)
			select {
			case errch <- err:
			default:
			}
			return err
		}
		// occasionally do double/triple uploads from the same payload
		for j := 0; j < 1; j++ {
			log.Printf("start cycle j=%d", j)
			if url, err = CheckedUpload(up, payload, dump || bp < 1); err != nil {
				log.Printf("CU err=%s", err)
				err = fmt.Errorf("error uploading: %s", err)
				if errch != nil {
					select {
					case errch <- err:
					default:
					}
				}
				return err
			}
			bp += payload.Length
			log.Printf("bp=%d", bp)
			select {
			case urlch <- url:
			default:
			}
			log.Printf("cycle end")
			if rand.Int()%5 == 0 {
				j--
			}
		}
		log.Printf("eor %d", i)

	}
	return nil
}

// uploads and checks (reads back data) right after the upload
func CheckedUpload(up Uploader, payload Payload, dump bool) (url string, err error) {
	if dump {
		log.Printf("Content-Type=%s", payload.ContentType)
	}
	// hr, ok := payload.Data.(HashedReader)
	// if !ok {
	// 	hr = NewHashedReader(payload.Data)
	// 	payload.Data = hr
	// }
	url, err = up.Upload(payload)
	if err != nil {
		return url, err
	}
	if url == "" {
		return url, fmt.Errorf("empty url!")
	}
	// uphash := hr.Sum()
	var r io.ReadCloser
	for i := 0; i < 10; i++ {
		if r, err = up.Get(url); err == nil {
			length, _, err := Hash(r)
			if err != nil {
				return url, err
			}
			if length != payload.Length {
				return url, fmt.Errorf("length mismatch for %s", url)
			}
			// if hashIsOk && !bytes.Equal(downhash, uphash) {
			// 	return url, fmt.Errorf("hash mismatch for %s (up=%x, down=%x)",
			// 		url, uphash, downhash)
			// }
			return url, nil
		}
		log.Printf("WARN[%d] cannot get %s: %s", i, url, err)
		time.Sleep(1 * time.Second)
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
	var (
		err  error
		resp *http.Response
		msg  string
	)
	for i := 0; i < 10; i++ {
		msg = ""
		if GzipOk {
			resp, err = http.Get(url)
		} else {
			req, e := http.NewRequest("GET", url, nil)
			if e == nil {
				req.Header.Set("Accept-Encoding", "ident")
				resp, err = http.DefaultClient.Do(req)
			} else {
				msg = fmt.Sprintf("cannot create request for %s: %s", url, e)
			}
		}
		if resp == nil {
			// return nil, fmt.Errorf("nil response for %s!", url)
			msg = fmt.Sprintf("nil response for %s!", url)
		} else {
			if err == nil {
				if 200 <= resp.StatusCode && resp.StatusCode <= 299 {
					return resp.Body, nil
				}
				msg = fmt.Sprintf("STATUS=%s (%s)", resp.Status, url)
			} else {
				// dumpResponse(resp, true)
				msg = fmt.Sprintf("erro with http.Get(%s): %s", url, err)
			}
		}
		log.Println(msg)
		time.Sleep(1 * time.Second)
	}
	return nil, errors.New(msg)
}

func (payload Payload) Post(url string) (respBody []byte, err error) {
	if payload.Length == 0 {
		err = errors.New("zero length payload!")
		return
	}
	reqbuf := bytes.NewBuffer(make([]byte, 0, payload.Length*2+256))
	formDataContentType, n, e := EncodePayload(reqbuf, bytes.NewReader(payload.Data),
		fmt.Sprintf("test-%d", payload.Length), payload.ContentType)
	if e != nil {
		err = e
		return
	}
	if n == 0 {
		err = errors.New("zero length encoded payload!")
		return
	}
	var (
		req  *http.Request
		resp *http.Response
	)
	req, e = http.NewRequest("POST", url, bytes.NewReader(reqbuf.Bytes()))
	if e != nil {
		err = fmt.Errorf("error creating POST to %s: %s", url, e)
		return
	}
	// log.Printf("CL=%d n=%d size=%d", req.ContentLength, n, len(reqbuf.Bytes()))
	req.ContentLength = int64(len(reqbuf.Bytes()))
	req.Header.Set("MIME-Version", "1.0")
	req.Header.Set("Content-Type", formDataContentType)
	if !GzipOk {
		req.Header.Set("Accept-Encoding", "ident")
	}

	resp, e = http.DefaultClient.Do(req)
	if e != nil {
		err = fmt.Errorf("error POSTing %+v: %s", req, e)
		return
	}
	req = resp.Request
	dumpRequest(req, false)
	if resp == nil || resp.Body == nil {
		err = fmt.Errorf("nil response")
		return
	}
	// if resp.ContentLength == -1 {
	// 	resp.ContentLength = 32
	// }
	defer resp.Body.Close()
	// dumpResponse(resp, e != nil)
	// log.Printf("resp.ContentLength=%d", resp.ContentLength)
	if resp.ContentLength > 0 {
		respBody = make([]byte, resp.ContentLength)
		if length, e := io.ReadFull(resp.Body, respBody); e == nil && length > 0 {
			respBody = respBody[:length]
		} else {
			err = fmt.Errorf("error reading response %d body: %s", length, e)
			return
		}
		log.Printf("CL=%d respBody=%s", resp.ContentLength, respBody)
	} else if resp.ContentLength < 0 {
		respBody, e = ioutil.ReadAll(resp.Body)
	}
	log.Printf("respBody=%s", respBody)
	if e != nil {
		err = fmt.Errorf("error reading response body: %s", e)
	}
	// resp.Body = ioutil.NopCloser(bytes.NewReader(respBody))

	if !(200 <= resp.StatusCode && resp.StatusCode <= 299) {
		err = fmt.Errorf("errorcode=%d message=%s", resp.StatusCode, respBody)
		return
	}

	return
}

func dumpRequest(req *http.Request, force bool) {
	if req != nil && (force || Dump) {
		buf, e := httputil.DumpRequestOut(req, true)
		if e != nil {
			log.Printf("!!! cannot dump request %v: %s", req, e)
		} else {
			log.Printf("\n>>>>>>\nrequest:\n%s", buf)
		}
	}
}

func dumpResponse(resp *http.Response, force bool) {
	if resp != nil && (force || Dump) {
		buf, e := httputil.DumpResponse(resp, true)
		if e != nil {
			log.Printf("!!! cannot dump response %v: %s", resp, e)
		} else {
			log.Printf("\n>>>>>>\nresponse:\n%s", buf)
		}
	}
}

// SubError
func SubError(err error, format string, args ...interface{}) error {
	args = append(args, args[0])
	args[0] = errors.New(strings.Replace(err.Error(), "\n", "\n  ", -1))
	return fmt.Errorf(format+":\n%s", args...)
}
