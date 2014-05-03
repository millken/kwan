package webfilter

import (
	"config"
	"github.com/millken/falcore"
	"net/http"
)

type StatusFilter int

func (s StatusFilter) FilterRequest(request *falcore.Request) *http.Response {
	vhost := request.Context["config"].(config.Vhost)
	if vhost.Status == 1 {
		return falcore.StringResponse(request.HttpRequest, 200, nil, "the site was paused!\n")
	}
	return nil
}
