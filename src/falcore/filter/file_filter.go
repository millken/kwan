package filter

import (
	"github.com/fitstar/falcore"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// A falcore RequestFilter for serving static files
// from the filesystem.  This filter returns nil if the file
// is not found or is outside of scope.
type FileFilter struct {
	// File system base path for serving files
	BasePath string
	// Prefix in URL path
	PathPrefix string
	// File to look for if path is a directory
	DirectoryIndex string
}

func (f *FileFilter) FilterRequest(req *falcore.Request) (res *http.Response) {
	// Clean asset path
	asset_path := filepath.Clean(filepath.FromSlash(req.HttpRequest.URL.Path))

	// Resolve PathPrefix
	if strings.HasPrefix(asset_path, f.PathPrefix) {
		asset_path = asset_path[len(f.PathPrefix):]
	} else {
		// The requested path doesn't fall into the scope of paths that are supposed to be handled by this filter
		return
	}

	// Resolve FSBase
	if f.BasePath != "" {
		asset_path = filepath.Join(f.BasePath, asset_path)
	} else {
		falcore.Error("file_filter requires a BasePath")
		return falcore.StringResponse(req.HttpRequest, 500, nil, "Server Error\n")
	}

	// Open File
	if file, err := os.Open(asset_path); err == nil {
		// If it's a directory, try opening the directory index
		if stat, err := file.Stat(); f.DirectoryIndex != "" && err == nil && stat.Mode()&os.ModeDir > 0 {
			file.Close()

			asset_path = filepath.Join(asset_path, f.DirectoryIndex)
			if file, err = os.Open(asset_path); err != nil {
				return
			}
		}

		// Make sure it's an actual file
		if stat, err := file.Stat(); err == nil && stat.Mode()&os.ModeType == 0 {
			res = &http.Response{
				Request:       req.HttpRequest,
				StatusCode:    200,
				Proto:         "HTTP/1.1",
				ProtoMajor:    1,
				ProtoMinor:    1,
				Body:          file,
				Header:        make(http.Header),
				ContentLength: stat.Size(),
			}
			if ct := mime.TypeByExtension(filepath.Ext(asset_path)); ct != "" {
				res.Header.Set("Content-Type", ct)
			}
		} else {
			file.Close()
		}
	} else {
		falcore.Finest("Can't open %v: %v", asset_path, err)
	}
	return
}
