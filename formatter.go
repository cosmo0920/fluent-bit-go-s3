package main

import (
	"bytes"
	"fmt"
	"os"
)
import "github.com/sirupsen/logrus"
import "golang.org/x/crypto/ssh/terminal"

const (
	ANSI_RESET   = "\033[0m"
	ANSI_BOLD    = "\033[1m"
	ANSI_CYAN    = "\033[96m"
	ANSI_MAGENTA = "\033[95m"
	ANSI_RED     = "\033[91m"
	ANSI_YELLOW  = "\033[93m"
	ANSI_BLUE    = "\033[94m"
	ANSI_GREEN   = "\033[92m"
	ANSI_WHITE   = "\033[97m"
)

type fluentBitLogFormat struct{}

//Format Specify logging format.
func (f *fluentBitLogFormat) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	bold_color := ANSI_BOLD
	reset_color := ANSI_RESET

	header_title := ""
	header_color := ""
	switch entry.Level {
	case logrus.TraceLevel:
		header_title = "trace"
		header_color = ANSI_BLUE
	case logrus.InfoLevel:
		header_title = "info"
		header_color = ANSI_GREEN
	case logrus.WarnLevel:
		header_title = "warn"
		header_color = ANSI_YELLOW
	case logrus.ErrorLevel:
		header_title = "error"
		header_color = ANSI_RED
	case logrus.DebugLevel:
		header_title = "debug"
		header_color = ANSI_YELLOW
	case logrus.FatalLevel:
		header_title = "fatal"
		header_color = ANSI_MAGENTA
	}

	if !terminal.IsTerminal(int(os.Stdout.Fd())) {
		header_color = ""
		bold_color = ""
		reset_color = ""
	}

	time := fmt.Sprintf("%s[%s%s%s]%s",
		bold_color, reset_color,
		entry.Time.Format("2006/01/02 15:04:05"),
		bold_color, reset_color)
	b.WriteString(time)

	level := fmt.Sprintf(" [%s%5s%s] ", header_color, header_title, reset_color)
	b.WriteString(level)

	if entry.Message != "" {
		b.WriteString(entry.Message)
	}

	if len(entry.Data) > 0 {
		b.WriteString(" || ")
	}
	for key, value := range entry.Data {
		b.WriteString(key)
		b.WriteByte('=')
		b.WriteByte('{')
		fmt.Fprint(b, value)
		b.WriteString("}, ")
	}

	b.WriteByte('\n')
	return b.Bytes(), nil
}
