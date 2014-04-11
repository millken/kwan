package filter

import (
	"bytes"
	"fmt"
	"github.com/fitstar/falcore"
	"io"
	"io/ioutil"
	"log"
	"mime"
	"net/http"
	"strings"
	"testing"
	"time"
)

var srv *falcore.Server

func init() {
	// Silence log output
	log.SetOutput(nil)

	// setup mime
	mime.AddExtensionType(".foo", "foo/bar")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".txt", "text/plain")
	mime.AddExtensionType(".png", "image/png")
	mime.AddExtensionType(".html", "text/html")

	go func() {

		// falcore setup
		pipeline := falcore.NewPipeline()
		pipeline.Upstream.PushBack(&FileFilter{
			PathPrefix:     "/",
			BasePath:       "../test/",
			DirectoryIndex: "index.html",
		})
		srv = falcore.NewServer(0, pipeline)
		if err := srv.ListenAndServe(); err != nil {
			panic(fmt.Sprintf("Could not start falcore: %v", err))
		}
	}()
}

func port() int {
	for srv.Port() == 0 {
		time.Sleep(1e7)
	}
	return srv.Port()
}

func get(p string) (r *http.Response, err error) {
	req, _ := http.NewRequest("GET", fmt.Sprintf("http://%v", fmt.Sprintf("localhost:%v/", port())), nil)
	req.URL.Path = p
	r, err = http.DefaultTransport.RoundTrip(req)
	return
}

var fourOhFourTests = []struct {
	name string
	url  string
}{
	{
		name: "basic invalid path",
		url:  "/this/path/doesnt/exist",
	},
	{
		name: "realtive pathing out of sandbox",
		url:  "/../README.md",
	},
	{
		name: "directory",
		url:  "/hello",
	},
}

func TestFourOhFour(t *testing.T) {
	for _, test := range fourOhFourTests {
		r, err := get(test.url)
		if err != nil {
			t.Errorf("%v Error getting file:", test.name, err)
			continue
		}
		if r.StatusCode != 404 {
			t.Errorf("%v Expected status 404, got %v", test.name, r.StatusCode)
		}
	}
}

var basicTests = []struct {
	name string
	mime string
	data []byte
	file string
	url  string
}{
	{
		name: "small text file",
		mime: "text/plain",
		data: []byte("Hello world!"),
		url:  "/hello/world.txt",
	},
	{
		name: "json file",
		mime: "application/json",
		file: "../test/foo.json",
		url:  "/foo.json",
	},
	{
		name: "png file",
		mime: "image/png",
		file: "../test/images/face.png",
		url:  "/images/face.png",
	},
	{
		name: "relative paths",
		mime: "application/json",
		file: "../test/foo.json",
		url:  "/images/../foo.json",
	},
	{
		name: "custom mime type",
		mime: "foo/bar",
		file: "../test/custom_type.foo",
		url:  "/custom_type.foo",
	},
	{
		name: "directory index",
		mime: "text/html",
		file: "../test/index.html",
		url:  "/",
	},
}

func TestBasicFiles(t *testing.T) {
	rbody := new(bytes.Buffer)
	for _, test := range basicTests {
		// read in test file data
		if test.file != "" {
			test.data, _ = ioutil.ReadFile(test.file)
		}

		r, err := get(test.url)
		if err != nil {
			t.Errorf("%v Error GETting file:%v", test.name, err)
			continue
		}
		if r.StatusCode != 200 {
			t.Errorf("%v Expected status 200, got %v", test.name, r.StatusCode)
			continue
		}
		if strings.Split(r.Header.Get("Content-Type"), ";")[0] != test.mime {
			t.Errorf("%v Expected Content-Type: %v, got '%v'", test.name, test.mime, r.Header.Get("Content-Type"))
		}
		rbody.Reset()
		io.Copy(rbody, r.Body)
		if rbytes := rbody.Bytes(); !bytes.Equal(test.data, rbytes) {
			t.Errorf("%v Body doesn't match.\n\tExpected:\n\t%v\n\tReceived:\n\t%v", test.name, test.data, rbytes)
		}
	}
}
