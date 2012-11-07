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

func Upload(baseUrl string, payload PLoad, dump bool) (key aostor.UUID, err error) {
	// log.Printf("body:\n%v", reqbuf)

	req, e := http.NewRequest("POST", baseUrl+"/up", bytes.NewReader(payload.encoded))
	if e != nil {
		err = e
		return
	}
	req.ContentLength = int64(len(payload.encoded))
	req.Header.Set("Content-Type", payload.ct)
	if dump {
		buf, e := httputil.DumpRequestOut(req, true)
		if e != nil {
			log.Panicf("cannot dump request %v: %s", req, e)
		} else {
			log.Printf("\n>>>>>>\nrequest:\n%v", buf)
		}
	}
	resp, e := http.DefaultClient.Do(req)
	if e != nil {
		buf, e := httputil.DumpRequestOut(req, true)
		if e != nil {
			log.Printf("cannot dump request %v: %s", req, e)
			return aostor.UUID{}, nil
		} else {
			log.Printf("\n>>>>>>\nrequest:\n%v", buf)
		}
	}
	if e != nil {
		err = e
		return
	}
	defer resp.Body.Close()
	buf := make([]byte, 32)
	n, e := resp.Body.Read(buf)
	if e != nil || dump {
		buf, e := httputil.DumpResponse(resp, true)
		if e != nil {
			log.Printf("cannot dump response %v: %s", resp, e)
		} else {
			log.Printf("\n<<<<<<\nresponse:\n%v", buf)
		}
	}
	if e == nil {
		key, e = aostor.UUIDFromBytes(buf[:n])
	}
	if e != nil {
		err = e
		return
	}
	if n != 2*aostor.UUIDLength || bytes.Equal(bytes.ToUpper(key[:3]), []byte{'E', 'R', 'R'}) {
		return aostor.UUID{}, fmt.Errorf("bad response: %s", key)
	}
	// log.Printf("%s", key)
	return
}

func Get(baseUrl string, key aostor.UUID, payload PLoad) (url string, err error) {
	url = baseUrl + "/" + key.String()
	resp, e := http.Get(url)
	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}
	if e != nil {
		err = fmt.Errorf("error with http.Get(%s): %s", url, e)
		return
	}
	if !(200 <= resp.StatusCode && resp.StatusCode <= 299) {
		err = fmt.Errorf("STATUS=%s (%s)", resp.Status, url)
		return
	}
	c := aostor.NewCounter()
	buf, err := ioutil.ReadAll(io.TeeReader(resp.Body, c))
	if err != nil {
		buf, e := httputil.DumpResponse(resp, true)
		if e != nil {
			log.Printf("cannot dump response %v: %s", resp, e)
		}
		err = fmt.Errorf("error reading from %s: %s\n<<<<<<\nresponse:\n%v",
			resp.Body, err, buf)
		return
	}
	if c.Num != payload.length {
		err = fmt.Errorf("length mismatch: read %d bytes (%d content-length=%d) for %s, required %d\n%s",
			c.Num, len(buf), resp.ContentLength, key, payload.length, resp)
		return
	}
	if payload.data != nil && uint64(len(payload.data)) == payload.length && !bytes.Equal(payload.data, buf) {
		err = fmt.Errorf("data mismatch: read %v, asserted %v", buf, payload.data)
		return
	}
	return
}
