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
	"bufio"
	"bytes"
	"io"
	"log"
	"mime/multipart"
	"os"
	"sync"
)

var (
	urandom      io.Reader
	payloadbuf   = bytes.NewBuffer(nil)
	payload_lock = sync.Mutex{}
)

type Payload struct {
	ContentType string
	Data        io.Reader
	Length      uint64
}

func getPayload(contentType string) (Payload, error) {
	payload_lock.Lock()
	defer payload_lock.Unlock()
	if payloadbuf == nil || urandom == nil {
		ur, err := os.Open("/dev/urandom")
		if err != nil {
			return Payload{}, err
		}
		payloadbuf = bytes.NewBuffer(nil)
		urandom = bufio.NewReader(ur)
	}
	n, err := io.CopyN(payloadbuf, urandom, 128)
	if err != nil {
		// payload_lock.Unlock()
		log.Panicf("cannot read %s: %s", urandom, err)
	}
	buf := payloadbuf.Bytes()
	length := len(buf)
	if length > 65 {
		payloadbuf.Write(buf[length-65:])
	}
	if length == 0 {
		log.Fatalf("zero payload (length=%d read=%d)", length, n)
	}
	if contentType == "" {
		contentType = "application/octet-strem"
	}
	return Payload{ContentType: contentType, Data: bytes.NewBuffer(buf),
		Length: uint64(length)}, nil
}

func EncodePayload(w io.Writer, r io.Reader, filename string) (int64, error) {
	mw := multipart.NewWriter(w)
	defer mw.Close()
	fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		log.Panicf("cannot create FormFile: %s", err)
	}
	return io.Copy(fw, r)
}
