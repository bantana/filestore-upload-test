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

// Aostor is the aostor instance
type Aostor struct {
	// BaseURL is the base url for aostor
	BaseURL string
}

// Upload uploads the payload
func (ao Aostor) Upload(payload Payload) (url string, err error) {
	respBody, e := payload.Post(ao.BaseURL + "/up")
	if e != nil {
		err = e
		return
	}
	//FIXME: urlencode
	return ao.BaseURL + "/" + string(respBody), nil
}

// Get gets the url
func (ao Aostor) Get(url string) (io.ReadCloser, error) {
	return GetURL(url)
}
