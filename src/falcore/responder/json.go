package responder

import (
	"bytes"
	"encoding/json"
	"github.com/fitstar/falcore"
	"net/http"
)

// Generate an http.Response by json encoding body using
// the standard library's json.Encoder.  error will be nil
// unless json encoding fails.
func JSONResponse(req *http.Request, status int, headers http.Header, body interface{}) (*http.Response, error) {
	buf := new(bytes.Buffer)
	if err := json.NewEncoder(buf).Encode(body); err != nil {
		return nil, err
	}

	if headers == nil {
		headers = make(http.Header)
	}
	if headers.Get("Content-Type") == "" {
		headers.Set("Content-Type", "application/json")
	}

	return falcore.SimpleResponse(req, status, headers, int64(buf.Len()), buf), nil
}
