package log

import (
	"bytes"
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/openconfig/goyang/pkg/indent"
	"github.com/sirupsen/logrus"
)

// ColorFormatter is a formatter that uses different colors for each level.
type ColorFormatter struct {
	// Indent defines by how much to indent each log line
	Indent int
}

var logColorMap = map[logrus.Level]*color.Color{
	logrus.PanicLevel: color.New(color.FgHiRed, color.Bold),
	logrus.FatalLevel: color.New(color.FgHiRed),
	logrus.ErrorLevel: color.New(color.FgRed),
	logrus.WarnLevel:  color.New(color.FgYellow),
	logrus.InfoLevel:  color.New(color.FgHiWhite),
	logrus.DebugLevel: color.New(color.FgCyan),
	logrus.TraceLevel: color.New(color.FgHiCyan),
}

var logEmojiMap = map[logrus.Level]string{
	logrus.PanicLevel: "😨 ",
	logrus.FatalLevel: "💥  ",
	logrus.ErrorLevel: "❌ ",
	logrus.WarnLevel:  "❗ ",
	logrus.InfoLevel:  "",
	logrus.DebugLevel: "",
	logrus.TraceLevel: "",
}

func (cf *ColorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	buff := &bytes.Buffer{}
	if entry.Buffer != nil {
		buff = entry.Buffer
	}

	if color.NoColor {
		fmt.Fprintf(buff, "%s%s\n", logEmojiMap[entry.Level], entry.Message)
	} else {
		reset := color.New(color.Reset)
		reset.Fprintf(buff, "")
		logColor, ok := logColorMap[entry.Level]
		if !ok {
			panic(fmt.Sprintf("unexpected level: %v", entry.Level))
		}
		logColor.Fprintf(buff, "%s%s", logEmojiMap[entry.Level], entry.Message)
		reset.Fprintf(buff, "")
		fmt.Fprintf(buff, "\n")
	}

	keys := []string{}
	for k := range entry.Data {
		keys = append(keys, k)
	}
	dataBuff := &bytes.Buffer{}
	sort.Strings(keys)
	for _, k := range keys {
		var i string
		if len(k) > 0 {
			fmt.Fprintf(dataBuff, "  %s:", k)
			i = "  "
		} else {
			i = ""
		}
		data := strings.TrimSuffix(fmt.Sprintf("%v", entry.Data[k]), "\n")
		if strings.Contains(data, "\n") {
			if len(k) > 0 {
				fmt.Fprintf(dataBuff, "\n")
			}
			fmt.Fprintf(dataBuff, "%s", indent.String(fmt.Sprintf("%s  ", i), data))
		} else {
			if len(k) > 0 {
				fmt.Fprintf(dataBuff, " ")
			}
			fmt.Fprintf(dataBuff, "  %s", data)
		}
	}
	if len(keys) > 0 {
		// FIXME if no color, remove color escape sequences from dataBuff
		fmt.Fprintf(buff, "%s", dataBuff.String())
		fmt.Fprintf(buff, "\n")
	}

	return indent.Bytes([]byte(strings.Repeat("  ", cf.Indent)), buff.Bytes()), nil
}
