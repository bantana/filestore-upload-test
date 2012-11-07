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

var (
	urandom      io.Reader
	payloadbuf   = bytes.NewBuffer(nil)
	payload_lock = sync.Mutex{}
)

type Payload struct {
	mimetype string
	data     io.Reader
	length   uint64
}

func getPayload() (PLoad, error) {
	payload_lock.Lock()
	defer payload_lock.Unlock()
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
	reqbuf := bytes.NewBuffer(make([]byte, 0, 2*length+256))
	mw := multipart.NewWriter(reqbuf)
	w, err := mw.CreateFormFile("upfile", fmt.Sprintf("test-%d", length))
	if err != nil {
		log.Panicf("cannot create FormFile: %s", err)
	}
	m, err := w.Write(buf)
	if err != nil {
		log.Printf("written payload is %d bytes (%s)", m, err)
	}
	mw.Close()
	return PLoad{buf, reqbuf.Bytes(),
		mw.FormDataContentType(), uint64(length)}, nil
}
