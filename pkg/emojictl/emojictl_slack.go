package emojictl

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gabriel-vasile/mimetype"
	"github.com/slack-go/slack"
)

const (
	slackHost         string = "slack.com"
	multipartBoundary string = "foo"
)

type slackHTTPRequest struct {
	Context context.Context
	Method  HTTPMethod
	Path    string
	Headers http.Header
	Body    io.ReadCloser
}

type slackHTTPResponse struct{ *http.Response }

func (s *slackHTTPResponse) JSON() (map[string]interface{}, error) {
	ct := s.Header.Get(string(ContentType))

	if strings.Split(ct, "; ")[0] != "application/json" {
		return nil, fmt.Errorf("body is not JSON: %s", ct)
	}

	parsed := map[string]interface{}{}
	if err := json.NewDecoder(s.Body).Decode(&parsed); err != nil {
		return nil, err
	}

	return parsed, nil
}

type slackHTTPClient struct {
	httpClient http.Client
	token      string
}

func (s *slackHTTPClient) defaultMultipartBody() MultipartBody {
	return MultipartBody(map[string]string{"token": s.token})
}

func (s *slackHTTPClient) Do(req *slackHTTPRequest) (*slackHTTPResponse, error) {
	boundary, mpBody, err := s.defaultMultipartBody().Render()
	if err != nil {
		return nil, err
	}

	res, err := s.httpClient.Do(func() *http.Request {
		return &http.Request{
			Method: string(req.Method),
			URL:    &url.URL{Scheme: "https", Host: slackHost, Path: req.Path},
			Header: func() http.Header {
				multipartContentTypeHeader := MakeMultiPartContentTypeHeaderValue(boundary)

				if req.Headers == nil {
					return HTTPHeaders{
						ContentType: multipartContentTypeHeader,
					}.ToMapStringSliceString()
				}

				if _, ok := req.Headers[string(ContentType)]; !ok {
					req.Headers.Set(string(ContentType), multipartContentTypeHeader)
				}

				return req.Headers
			}(),
			Body: func() io.ReadCloser {
				if req.Body == nil {
					return mpBody
				}

				return req.Body
			}(),
		}
	}().WithContext(req.Context))

	return &slackHTTPResponse{res}, err
}

type SlackEmojictl struct {
	slackAPIClient  *slack.Client
	slackHTTPClient *slackHTTPClient
}

func NewSlackEmojictl(token string) (*SlackEmojictl, error) {
	return &SlackEmojictl{
		slack.New(token), &slackHTTPClient{*http.DefaultClient, token},
	}, nil
}

func (s *SlackEmojictl) makeEmojiMultipartBody(e *Emoji) (string, io.ReadCloser, error) {
	eBytes, err := e.Get()
	if err != nil {
		return "", nil, err
	}

	fName := filepath.Base(e.Location.Path)
	mpBody := new(bytes.Buffer)
	mpWriter := multipart.NewWriter(mpBody)

	for k, v := range map[string]string{
		"mode":  "data",
		"name":  e.Name,
		"token": s.slackHTTPClient.token,
	} {
		if err := mpWriter.WriteField(k, v); err != nil {
			return "", nil, err
		}
	}

	imageBody, err := mpWriter.CreatePart(textproto.MIMEHeader(HTTPHeaders{
		ContentDisposition: fmt.Sprintf("form-data; name=image; filename=\"%s\"", fName),
		ContentType:        mimetype.Detect(eBytes).String(),
	}.ToMapStringSliceString()))
	if err != nil {
		return "", nil, err
	}

	if _, err := imageBody.Write(eBytes); err != nil {
		return "", nil, err
	}

	if err := mpWriter.Close(); err != nil {
		return "", nil, err
	}

	return mpWriter.Boundary(), ioutil.NopCloser(mpBody), nil
}

func (s *SlackEmojictl) ListEmojis(ctx context.Context) ([]*Emoji, error) {
	res, err := s.slackHTTPClient.Do(&slackHTTPRequest{
		Context: ctx, Method: POST, Path: "/api/emoji.list",
	})
	if err != nil {
		return nil, err
	}

	resJSON, err := res.JSON()
	if err != nil {
		return nil, err
	}

	emojis := []*Emoji{}
	for name, image := range resJSON["emoji"].(map[string]interface{}) {
		u, err := url.Parse(image.(string))
		if err != nil {
			return nil, err
		}
		emojis = append(emojis, &Emoji{name, u})
	}

	// sort alphabetically
	sort.Slice(emojis, func(i, j int) bool {
		return int(rune(emojis[i].Name[0])) < int(rune(emojis[j].Name[0]))
	})

	return emojis, nil
}

func (s *SlackEmojictl) AddEmoji(ctx context.Context, e *Emoji) error {
	bound, mpBody, err := s.makeEmojiMultipartBody(e)
	if err != nil {
		return err
	}

	res, err := s.slackHTTPClient.Do(&slackHTTPRequest{
		Context: ctx,
		Method:  POST,
		Path:    "/api/emoji.add",
		Headers: http.Header(HTTPHeaders{
			ContentType: MakeMultiPartContentTypeHeaderValue(bound),
		}.ToMapStringSliceString()),
		Body: mpBody,
	})
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("non-200 response: %+v", res)
	}

	resJSON, err := res.JSON()
	if err != nil {
		return err
	}

	if !resJSON["ok"].(bool) {
		return fmt.Errorf("upload failed: %s", resJSON["error"])
	}

	return nil
}

func (s *SlackEmojictl) RemoveEmoji(ctx context.Context, e *Emoji) error {
	bound, body, err := MultipartBody(map[string]string{
		"name":  e.Name,
		"token": s.slackHTTPClient.token,
	}).Render()
	if err != nil {
		return err
	}

	res, err := s.slackHTTPClient.Do(&slackHTTPRequest{
		Context: ctx,
		Method:  POST,
		Headers: http.Header(HTTPHeaders{
			ContentType: MakeMultiPartContentTypeHeaderValue(bound),
		}.ToMapStringSliceString()),
		Path: "/api/emoji.remove",
		Body: ioutil.NopCloser(body),
	})
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("non-200 response: %+v", res)
	}

	return nil
}

func (s *SlackEmojictl) AliasEmoji(ctx context.Context, src, dest string) error {
	bound, body, err := MultipartBody(map[string]string{
		"mode":      "alias",
		"name":      src,
		"alias_for": fmt.Sprintf(":%s:", dest),
		"token":     s.slackHTTPClient.token,
	}).Render()
	if err != nil {
		return err
	}

	res, err := s.slackHTTPClient.Do(&slackHTTPRequest{
		Context: ctx,
		Method:  POST,
		Path:    "/api/emoji.add",
		Headers: http.Header(HTTPHeaders{
			ContentType: MakeMultiPartContentTypeHeaderValue(bound),
		}.ToMapStringSliceString()),
		Body: ioutil.NopCloser(body),
	})
	if err != nil {
		return err
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("non-200 response: %+v", res)
	}

	resJSON, err := res.JSON()
	if err != nil {
		return err
	}

	if !resJSON["ok"].(bool) {
		return fmt.Errorf("alias failed: %s", resJSON["error"])
	}

	return nil
}

var _ Emojictl = (*SlackEmojictl)(nil)
