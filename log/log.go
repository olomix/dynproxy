package log

import (
	"flag"
	"fmt"
	"github.com/mgutz/ansi"
	golog "log"
	"log/syslog"
	"os"
)

var debug = flag.Bool("d", false, "turn on debug info")
var isSyslog = flag.Bool("syslog", false, "send logs to Syslog")
var syslogTag = flag.String("tag", "", "Syslog tag")

var error_prefix = ansi.Color("ERROR ", "red")
var debug_prefix = ansi.Color("DEBUG ", "yellow")
var info_prefix = ansi.Color("INFO  ", "green")
var logflags = golog.Ldate | golog.Ltime | golog.Lshortfile
var infolog = golog.New(os.Stderr, info_prefix, logflags)
var errorlog = golog.New(os.Stderr, error_prefix, logflags)
var debuglog = golog.New(os.Stderr, debug_prefix, logflags)
var sysinfolog *golog.Logger
var syserrorlog *golog.Logger
var sysdebuglog *golog.Logger

func Print(args ...interface{}) {
	infolog.Output(2, fmt.Sprint(args...))
	if sysinfolog != nil {
		sysinfolog.Output(2, fmt.Sprint(args...))
	}
}

func Printf(format string, args ...interface{}) {
	infolog.Output(2, fmt.Sprintf(format, args...))
	if sysinfolog != nil {
		sysinfolog.Output(2, fmt.Sprintf(format, args...))
	}
}

func Error(args ...interface{}) {
	errorlog.Output(2, fmt.Sprint(args...))
	if syserrorlog != nil {
		syserrorlog.Output(2, fmt.Sprint(args...))
	}
}

func Errorf(format string, args ...interface{}) {
	errorlog.Output(2, fmt.Sprintf(format, args...))
	if syserrorlog != nil {
		syserrorlog.Output(2, fmt.Sprintf(format, args...))
	}
}

func Debug(args ...interface{}) {
	if *debug {
		debuglog.Output(2, fmt.Sprint(args...))
		if sysdebuglog != nil {
			sysdebuglog.Output(2, fmt.Sprint(args...))
		}
	}
}

func Debugf(format string, args ...interface{}) {
	if *debug {
		debuglog.Output(2, fmt.Sprintf(format, args...))
		if sysdebuglog != nil {
			sysdebuglog.Output(2, fmt.Sprintf(format, args...))
		}
	}
}

func IsDebug() bool {
	return *debug
}

func SetupLogs() (err error) {
	golog.SetFlags(logflags)
	if !*isSyslog {
		return nil
	}

	sysinfolog, err = NewSysLogger(syslog.LOG_LOCAL4|syslog.LOG_NOTICE, 0)
	if err != nil {
		return err
	}
	syserrorlog, err = NewSysLogger(syslog.LOG_LOCAL4|syslog.LOG_CRIT, 0)
	if err != nil {
		return err
	}
	sysdebuglog, err = NewSysLogger(syslog.LOG_LOCAL4|syslog.LOG_DEBUG, 0)
	return err
}

func NewSysLogger(p syslog.Priority, logFlag int) (*golog.Logger, error) {
	s, err := syslog.New(p, *syslogTag)
	if err != nil {
		return nil, err
	}
	return golog.New(s, "", logFlag), nil
}
