package ggen

import (
	"fmt"
	"io"

	"github.com/iolivernguyen/ggen/ggen/logging"
)

type Logger = logging.Logger
type LogAttr = logging.Attr
type LogLevel = logging.Level
type LogHandler = logging.Handler

const (
	DebugLevel = logging.DebugLevel
	InfoLevel  = logging.InfoLevel
	WarnLevel  = logging.WarnLevel
	ErrorLevel = logging.ErrorLevel
)

var logger Logger

type embededLogger struct {
	Logger
}

type defaultLogHandler struct {
	w     io.Writer
	level logging.Level
	attrs []logging.Attr
}

func (h defaultLogHandler) Enabled(level logging.Level) bool {
	return level >= h.level
}

func (h defaultLogHandler) WithAttrs(attrs []logging.Attr) logging.Handler {
	newAttrs := make([]logging.Attr, 0, len(h.attrs)+len(attrs))
	newAttrs = append(newAttrs, h.attrs...)
	newAttrs = append(newAttrs, attrs...)
	h.attrs = newAttrs
	return h // copy
}

func (h defaultLogHandler) Handle(r logging.Record) (err error) {
	debugEnabled := h.Enabled(-1)
	if debugEnabled {
		_, err = fmt.Fprintf(h.w, "%7s: %s", r.Level, r.Message)
	} else if r.Level != 0 {
		_, err = fmt.Fprintf(h.w, "%s", r.Message)
	} else {
		_, err = fmt.Fprint(h.w, r.Message)
	}
	if err != nil {
		return err
	}
	for _, attr := range h.attrs {
		fmt.Fprintf(h.w, " %s=%v", attr.Key, attr.Value)
	}
	r.Attrs(func(attr logging.Attr) {
		fmt.Fprintf(h.w, " %s=%q", attr.Key, attr.Value)
	})
	_, err = fmt.Fprintf(h.w, "\n")
	return err
}
