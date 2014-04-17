package structs

import(
  "net/http"
  "io/ioutil"
  "log"
)

type CachedResponse struct {
  StatusCode int
  ContentLength int64
  Headers http.Header
  Body []byte
}

func NewCachedResponse(res *http.Response) (response *CachedResponse) {
  body, err := ioutil.ReadAll(res.Body)
  if err != nil {
     log.Fatal("Error reading from Body", err)
  }

  response = &CachedResponse{
    Body: body, 
    Headers: res.Header, 
    StatusCode: res.StatusCode,
    ContentLength: res.ContentLength,
  }
  
  return
}