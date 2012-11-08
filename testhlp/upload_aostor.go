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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httputil"
)

type Aostor struct {
	BaseUrl string
}

func (ao Aostor) Upload(payload Payload) (url string, err error) {
	var n int64
	reqbuf := bytes.NewBuffer(nil)
	n, err = EncodePayload(reqbuf, payload.Data, fmt.Sprintf("test-%d", payload.Length))
	if err != nil {
		return
	}
	req, e := http.NewRequest("POST", ao.BaseUrl+"/up", reqbuf)
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
			return "", err
		} else {
			log.Printf("\n>>>>>>\nrequest:\n%v", buf)
		}
	}
	if e != nil {
		err = e
		return
	}
	defer resp.Body.Close()
	var buf []byte
	if resp.ContentLength > 0 {
		buf = make([]byte, resp.ContentLength)
		length, e := io.ReadFull(resp.Body, buf)
		if e != nil {
			err = e
			return
		}
		buf = buf[:length]
	} else {
		buf, e = ioutil.ReadAll(resp.Body)
	}

	if e != nil { //|| dump {
		err = e
		buf, e := httputil.DumpResponse(resp, true)
		if e != nil {
			log.Printf("cannot dump response %v: %s", resp, e)
		} else {
			log.Printf("\n<<<<<<\nresponse:\n%v", buf)
		}
		return
	}
	//FIXME: urlencode
	return ao.BaseUrl + "/" + string(buf), nil
}

func (ao Aostor) Get(url string) (io.ReadCloser, error) {
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
