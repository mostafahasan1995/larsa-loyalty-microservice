package util

import (
	"bytes"
	"net/http"
)

type ResponseWriterWrapper struct {
	w          *http.ResponseWriter
	Body       *bytes.Buffer
	StatusCode *int
}

func NewResponseWriterWrapper(w http.ResponseWriter) ResponseWriterWrapper {
	var buf bytes.Buffer
	var statusCode int = 200
	return ResponseWriterWrapper{
		w:          &w,
		Body:       &buf,
		StatusCode: &statusCode,
	}
}

func (rww ResponseWriterWrapper) Write(buf []byte) (int, error) {
	rww.Body.Write(buf)
	return (*rww.w).Write(buf)
}

func (rww ResponseWriterWrapper) Header() http.Header {
	return (*rww.w).Header()

}

func (rww ResponseWriterWrapper) WriteHeader(statusCode int) {
	(*rww.StatusCode) = statusCode
	(*rww.w).WriteHeader(statusCode)
}
