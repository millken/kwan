package core

import (
	"container/list"
	"net/http"
	"logger"
)

type Pipeline struct {
	Upstream   *list.List
	Downstream *list.List
}

var pipeline = NewPipeline()

func AddRequestFilter(reqfilter interface{}) {
	pipeline.Upstream.PushBack(reqfilter)
}

func AddResponseFilter(resfilter *ResponseFilter) {
	pipeline.Upstream.PushBack(resfilter)
}

func NewPipeline() (l *Pipeline) {
	l = new(Pipeline)
	l.Upstream = list.New()
	l.Downstream = list.New()
	return
}

// Pipelines are valid RequestFilters.  This makes them nestable.
func (p *Pipeline) FilterRequest(req *Request) *http.Response {
	return p.execute(req)
}

func (p *Pipeline) execute(req *Request) (res *http.Response) {
	for e := p.Upstream.Front(); e != nil && res == nil; e = e.Next() {
		switch filter := e.Value.(type) {
		case RequestFilter:
			res = p.execFilter(req, filter)
			if res != nil {
				break
			}
		default:
			logger.Error("%v (%T) is not a RequestFilter\n", e.Value, e.Value)
			break
		}
	}

	if res != nil {
		p.down(req, res)
	}

	return
}

func (p *Pipeline) execFilter(req *Request, filter RequestFilter) *http.Response {

	return filter.FilterRequest(req)
}

func (p *Pipeline) down(req *Request, res *http.Response) {
	for e := p.Downstream.Front(); e != nil; e = e.Next() {
		if filter, ok := e.Value.(ResponseFilter); ok {
			filter.FilterResponse(req, res)
		} else {
			// TODO
			break
		}
	}
}
