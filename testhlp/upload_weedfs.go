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
	"io"
)

type Weed struct {
	MasterUrl string
}

func (we Weed) Upload(payload Payload) (url string, err error) {
	r, e := GetUrl(we.MasterUrl + "/dir/assign")
	if e != nil {
		err = e
		return
	}
	//read JSON

	//FIXME: urlencode
	return "", nil
}

func (we Weed) Get(url string) (io.ReadCloser, error) {
	return GetUrl(url)
}
