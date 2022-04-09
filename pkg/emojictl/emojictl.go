package emojictl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
)

// Emojictl descibes an interface to manage emojis.
type Emojictl interface {
	ListEmojis(context.Context) ([]*Emoji, error)
	AddEmoji(context.Context, *Emoji) error
	RemoveEmoji(context.Context, *Emoji) error
	AliasEmoji(context.Context, string, string) error
}

// HTTPMethod represents an HTTP request method
type HTTPMethod string

const (
	GET     HTTPMethod = "GET"
	HEAD    HTTPMethod = "HEAD"
	POST    HTTPMethod = "POST"
	PUT     HTTPMethod = "PUT"
	DELETE  HTTPMethod = "DELETE"
	CONNECT HTTPMethod = "CONNECT"
	OPTIONS HTTPMethod = "OPTIONS"
	TRACE   HTTPMethod = "TRACE"
	PATCH   HTTPMethod = "PATCH"
)

// HTTPHeader represents an HTTP request or response header key
type HTTPHeader string

const (
	ContentType        HTTPHeader = "Content-Type"
	ContentDisposition HTTPHeader = "Content-Disposition"
)

// All supported HTTP headers in this project
type HTTPHeaders struct {
	ContentDisposition string `header:"Content-Disposition"`
	ContentType        string `header:"Content-Type"`
	Cookie             string `header:"Cookie"`
}

// ToMapStringSliceString is a helper to make HTTPHeaders compatible with
// stdlib types like http.Header and textproto.MIMEHeader
func (h HTTPHeaders) ToMapStringSliceString() map[string][]string {
	res := map[string][]string{}
	t := reflect.TypeOf(h)
	v := reflect.ValueOf(h)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		res[f.Tag.Get("header")] = []string{v.Field(i).Interface().(string)}
	}

	return res
}

// MultipartBody is a declarative representation of a multipart-encoded HTTP
// request body
type MultipartBody map[string]string

// Render materializes the multipart request body from the supplied
// map[string]string. It returns the boundary as a string as well as an
// io.ReadCloser that contains the body bytes.
func (m MultipartBody) Render() (string, io.ReadCloser, error) {
	body := new(bytes.Buffer)
	mpWriter := multipart.NewWriter(body)
	defer mpWriter.Close()

	for k, v := range m {
		if err := mpWriter.WriteField(k, v); err != nil {
			return "", nil, err
		}
	}

	return mpWriter.Boundary(), ioutil.NopCloser(body), nil
}

// Given a multipart form boundary, returns the header value for a multipart
// Content-Type header.
func MakeMultiPartContentTypeHeaderValue(boundary string) string {
	return fmt.Sprintf("multipart/form-data; boundary=%s", boundary)
}

// Get a filename (without its extension if it has one) from a URI path.
// Includes handling filesystem paths (as they are a form of URI).
func FilenameNoExt(path string) string {
	filename := filepath.Base(path)
	return filename[0 : len(filename)-len(filepath.Ext(filename))]
}

// Emoji represents an emoji.
type Emoji struct {
	Name     string
	Location *url.URL
}

// Get retrieves the emoji data depending on its source as specified in the URI.
func (e *Emoji) Get() ([]byte, error) {
	var dataReader io.Reader

	switch e.Location.Scheme {
	case "file":
		fd, err := os.Open(e.Location.Path)
		if err != nil {
			return nil, err
		}
		defer fd.Close()

		dataReader = fd
	case "http", "https":
		res, err := http.Get(e.Location.String())
		if err != nil {
			return nil, err
		}

		dataReader = res.Body
	default:
		return nil, fmt.Errorf("unsupported source: %s", e.Location.Scheme)
	}

	return ioutil.ReadAll(dataReader)
}
