// Package lg implements a simple verbosity logger interface. This follows the design of golang.org/x/exp/slog.
//
// The default verbosity is 0, and can be changed by setting the environment variable GGEN_LOGGING. For example:
//
//	GGEN_LOGGING=0          : set the current verbosity to 0 (default)
//
//	gglog.Debug("hello")       : will not be printed
//	gglog.Info("hello")        : will print "hello"
//
// User of ggen package can replace the default logger with their own implementation by implementing the Logger
// interface.
package gglog

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"

	"golang.org/x/exp/slog"
)

const EnvLogging = "GGEN_LOGGING"

type Level = slog.Level

var _ Logger = slog.Logger{}

// Type Logger exposes the logging interface from golang.org/x/exp/slog. User of ggen package can replace the default
// logger with their own implementation by implementing the Logger interface.
type Logger interface {

	// Enabled reports whether logger emits log records at the given level.
	Enabled(level Level) bool

	// Log emits a log record with the current time and the given level and message.
	Log(level Level, msg string, args ...any)

	// Debug calls Logger.Debug on the default logger.
	Debug(msg string, args ...any)

	// Warn logs at InfoLevel.
	Info(msg string, args ...any)

	// Warn logs at WarnLevel.
	Warn(msg string, args ...any)

	// Error logs at ErrorLevel. If err is non-nil, Error appends Any("err", err) to the list of attributes.
	Error(msg string, err error, args ...any)
}

var defaultLogger Logger
var lock sync.RWMutex

// L return the default logger. It can be overwritten by SetLogger().
func L() Logger {
	lock.RLock()
	defer lock.RUnlock()

	if defaultLogger == nil {
		defaultLogger = newDefaultLogger()
	}
	return defaultLogger
}

// Enabled calls Logger.Enabled on the default logger.
func Enabled(level Level) bool {
	return L().Enabled(level)
}

// Log calls Logger.Log on the default logger.
func Log(level Level, msg string, args ...any) {
	L().Log(level, msg, args...)
}

// Debug calls Logger.Debug on the default logger.
func Debug(msg string, args ...any) {
	L().Debug(msg, args...)
}

// Info calls Logger.Info on the default logger.
func Info(msg string, args ...any) {
	L().Info(msg, args...)
}

// Warn calls Logger.Warn on the default logger.
func Warn(msg string, args ...any) {
	L().Warn(msg, args...)
}

// Error calls Logger.Error on the default logger.
func Error(msg string, err error, args ...any) {
	L().Error(msg, err, args...)
}

// SetLogger is used to overwrite the default logger with a custom implementation. It must be called before any log is
// created.
func SetLogger(logger Logger) {
	lock.Lock()
	defer lock.Unlock()

	if defaultLogger != nil {
		panic("logger is already set")
	}
	defaultLogger = logger
}

type DefaultHandler struct {
	w     io.Writer
	level Level
	group string
	attrs []slog.Attr
}

func NewDefaultHandler(w io.Writer, level Level) slog.Handler {
	return &DefaultHandler{w: w, level: level}
}

func (h DefaultHandler) Enabled(level slog.Level) bool {
	return level >= h.level
}

func (h DefaultHandler) Handle(r slog.Record) (err error) {
	_, err = fmt.Fprintf(h.w, "%7s: %s", r.Level, r.Message)
	if err != nil {
		return err
	}
	r.Attrs(func(attr slog.Attr) {
		_, _ = fmt.Fprintf(h.w, " %s=%q", attr.Key, attr.Value)
	})
	_, err = fmt.Fprintln(h.w)
	return err
}

func (h DefaultHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &DefaultHandler{w: h.w, level: h.level, attrs: attrs, group: h.group}
}

func (h DefaultHandler) WithGroup(name string) slog.Handler {
	return &DefaultHandler{w: h.w, level: h.level, attrs: h.attrs, group: name}
}

func newDefaultLogger() Logger {
	verbosity := 0
	loggingEnv := os.Getenv(EnvLogging)
	if loggingEnv != "" {
		v, err := strconv.Atoi(loggingEnv)
		if err != nil {
			msg := fmt.Sprintf("failed to parse %s environment variable, default to 0", EnvLogging)
			slog.Log(slog.ErrorLevel, msg)
		}
		verbosity = v
	}

	handler := NewDefaultHandler(os.Stderr, slog.Level(verbosity))
	return slog.New(handler)
}
