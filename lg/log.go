// lg implements a simple verbosity logger interface.
//
// The default verbosity is 0, and can be changed by setting the environment variable GGEN_LOGGING. For example:
//
//     GGEN_LOGGING=1          : set the current verbosity to 1
//
//     logger.V(0).Printf(...) : print log, since V(0) >  GGEN_LOGGING
//     logger.V(1).Printf(...) : print log, since V(1) == GGEN_LOGGING
//     logger.V(2).Printf(...) : not print, since V(2) >  GGEN_LOGGING
//
// User of ggen package can replace the default logger with their own implementation by implementing the Logger
// interface.
package lg

import (
	"log"
	"os"
	"strconv"
	"sync"
)

type Logger interface {

	// Verbosed checks if the current verbosity level is equal or higher than the param.
	Verbosed(verbosity int) bool

	// V returns a VerbosedLogger, which only outputs log when the current verbosity level is equal or higher than the
	// log line verbosity.
	V(verbosity int) VerbosedLogger

	// GetV returns the current verbosity.
	GetV() int

	// SetV sets a new verbosity, and return the last verbosity. The new verbosity must be equal or larger than the
	// current verbosity.
	SetV(verbosity int) int
}

type VerbosedLogger interface {
	Print(args ...interface{})
	Printf(format string, args ...interface{})
	Println(args ...interface{})
}

var _ Logger = logger(0)
var _ VerbosedLogger = logger(0)

var New func() Logger
var verbosity int
var lock sync.RWMutex

func init() {
	lock.Lock()
	defer lock.Unlock()

	New = newLogger
	verbosity, _ = strconv.Atoi(os.Getenv("GGEN_LOGGING"))
}

type logger int

func (_ logger) Verbosed(v int) bool {
	lock.RLock()
	defer lock.RUnlock()
	return v <= verbosity
}

func (l logger) V(verbosity int) VerbosedLogger {
	return logger(verbosity)
}

func (_ logger) GetV() int {
	lock.RLock()
	defer lock.RUnlock()

	return verbosity
}

func (_ logger) SetV(v int) (last int) {
	lock.Lock()
	defer lock.Unlock()

	last, verbosity = verbosity, v
	return last
}

func (l logger) Printf(format string, args ...interface{}) {
	lock.RLock()
	ok := int(l) <= verbosity
	lock.RUnlock()

	if ok {
		log.Printf(format, args...)
	}
}

func (l logger) Print(args ...interface{}) {
	lock.RLock()
	ok := int(l) <= verbosity
	lock.RUnlock()

	if ok {
		log.Print(args...)
	}
}

func (l logger) Println(args ...interface{}) {
	lock.RLock()
	ok := int(l) <= verbosity
	lock.RUnlock()

	if ok {
		log.Println(args...)
	}
}

func newLogger() Logger {
	return logger(0)
}
