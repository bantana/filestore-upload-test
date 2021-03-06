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
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/textproto"
	"os"
	"strings"
	"sync"
)

var (
	payloadbuf []byte
	pos, size  int
	//PayloadSizeInit is the initial payload size
	PayloadSizeInit = 1 << 15
	//PayloadSizeMax is the maximum payload size
	PayloadSizeMax = 1 << 20
	//PayloadSizeStep is the payload size increase step
	PayloadSizeStep = 1 << 15
	payloadLock     = sync.Mutex{}

	//Compressable says whether the payload should be compressable or not
	Compressable = false
	multiplier   = 1
)

// Payload is one mail part
type Payload struct {
	ContentType string
	// Data        io.Reader
	Data   []byte
	Length uint64
}

func getPayload(contentType string) (Payload, error) {
	payloadLock.Lock()
	defer payloadLock.Unlock()
	if payloadbuf == nil {
		if PayloadSizeMax < PayloadSizeInit {
			PayloadSizeMax = PayloadSizeInit * 2
		}
		ur, err := os.Open("/dev/urandom")
		if err != nil {
			return Payload{}, err
		}
		payloadbuf = make([]byte, PayloadSizeMax)
		length := len(payloadbuf)
		if Compressable {
			for multiplier = 1 << 8; multiplier > 1; multiplier >>= 1 {
				if length > multiplier {
					break
				}
			}
			length /= multiplier
		}
		log.Printf("compressable? %s => multiplier=%d, length=%d", Compressable, multiplier, length)
		if n, err := io.ReadFull(ur, payloadbuf[:length]); err != nil || n != length {
			log.Panicf("cannot read %d bytes from /dev/urandom, just %d: %s",
				length, n, err)
		}
		if multiplier > 1 {
			for i := length - 1; i > 0; i-- {
				for j := 0; j < multiplier; j++ {
					payloadbuf[i*multiplier+j] = payloadbuf[i+j]
				}
			}
		}
		size = PayloadSizeInit
	}
	buf := payloadbuf[pos : pos+size]
	if Debug {
		log.Printf("pos=%d size=%d", pos, size)
	}
	if pos+size < len(payloadbuf)-1 {
		pos++
	} else {
		pos = 0
		if size < len(payloadbuf)-1 {
			size += PayloadSizeStep
		} else {
			size = PayloadSizeInit
		}
	}

	length := len(buf)
	if length == 0 {
		log.Fatalf("zero payload")
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return Payload{ContentType: contentType, Data: buf, Length: uint64(length)}, nil
}

// EncodePayload encodes the payload
func EncodePayload(w io.Writer, r io.Reader, filename, contentType string) (string, int64, error) {
	mw := multipart.NewWriter(w)
	defer mw.Close()
	fw, err := CreateFormFile(mw, "file", filename, contentType)
	// fw, err := mw.CreateFormFile("file", filename)
	if err != nil {
		log.Panicf("cannot create FormFile: %s", err)
	}
	n, err := io.Copy(fw, r)
	return mw.FormDataContentType(), n, err
}

// CreateFormFile creates a form file
func CreateFormFile(w *multipart.Writer, fieldname, filename, contentType string) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Type", contentType)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(fieldname), escapeQuotes(filename)))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return w.CreatePart(h)
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}
