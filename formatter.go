package main

import (
    "bytes"
    "fmt"
)
import "github.com/sirupsen/logrus"

type fluentBitLogFormat struct {}

//Format Specify logging format.
func (f *fluentBitLogFormat) Format(entry *logrus.Entry) ([]byte, error) {
	var b *bytes.Buffer

	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	b.WriteByte('[')
	b.WriteString(entry.Time.Format("2006/01/02 15:04:05"))
	b.WriteString("]")

	l := fmt.Sprintf(" [%5s] ", entry.Level.String())
	b.WriteString(l)

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
