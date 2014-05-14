package utils

import (
	"io/ioutil"
	"net/http"
)

type CachedResponse struct {
	StatusCode    int
	ContentLength int64
	Headers       http.Header
	Body          []byte
}

func NewCachedResponse(res *http.Response) (*CachedResponse, error) {
	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	response := &CachedResponse{
		Body:          body,
		Headers:       res.Header,
		StatusCode:    res.StatusCode,
		ContentLength: res.ContentLength,
	}

	return response, nil
}
