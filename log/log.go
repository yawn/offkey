package log

import (
	"fmt"
	"io"
	"os"
)

type Log struct {
	password int
	w        io.Writer
}

func New() *Log {

	return &Log{
		w: os.Stdout,
	}

}

func (l *Log) Err(msg string, args ...interface{}) {

	l.Msg(msg, args...)
	os.Exit(1)

}

func (l *Log) Msg(msg string, args ...interface{}) {

	if l.password > 0 {
		fmt.Fprintf(l.w, fmt.Sprintf("\r%%-%ds", l.password), fmt.Sprintf(msg, args...))
		l.password = 0
	} else {
		fmt.Fprintf(l.w, "%s\n", fmt.Sprintf(msg, args...))
	}

}

func (l *Log) Password(msg string, args ...interface{}) {

	m := fmt.Sprintf(msg, args...)
	fmt.Fprintf(l.w, fmt.Sprintf("\r%%-%ds", l.password), m)

	l.password = len(m)

}
