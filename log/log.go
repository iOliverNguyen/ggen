// log implements a simple verbosity logger interface. User of ggen package can replace the default logger with their own version of logger.
package log

import (
	"log"
	"os"
	"strconv"
)

type Logger interface {
	Verbosed(verbosity int) bool
	V(verbosity int) VerbosedLogger
}

type VerbosedLogger interface {
	Printf(format string, args ...interface{})
}

var New func() Logger

var verbosity int

func init() {
	New = newLogger
	verbosity, _ = strconv.Atoi(os.Getenv("LOGGING"))
}

type logger int

func (l logger) V(verbosity int) VerbosedLogger {
	return logger(verbosity)
}

func (_ logger) Verbosed(v int) bool {
	return v <= verbosity
}

func (l logger) Printf(format string, args ...interface{}) {
	if int(l) <= verbosity {
		log.Printf(format, args...)
	}
}

func newLogger() Logger {
	return logger(0)
}
