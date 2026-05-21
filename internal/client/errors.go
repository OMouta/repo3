package client

import (
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
)

type Error struct {
	StatusCode int
	Code       string
	Message    string
	Key        string
}

func (e *Error) Error() string {
	if e.Code == "" {
		return fmt.Sprintf("repo3 server returned status %d", e.StatusCode)
	}
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func IsCode(err error, code string) bool {
	e, ok := err.(*Error)
	return ok && e.Code == code
}

func errorFromResponse(resp *http.Response) error {
	defer resp.Body.Close()

	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var parsed struct {
		XMLName xml.Name `xml:"Error"`
		Code    string   `xml:"Code"`
		Message string   `xml:"Message"`
		Key     string   `xml:"Key"`
	}
	if len(data) > 0 {
		_ = xml.Unmarshal(data, &parsed)
	}
	return &Error{
		StatusCode: resp.StatusCode,
		Code:       parsed.Code,
		Message:    parsed.Message,
		Key:        parsed.Key,
	}
}
