package log

import (
	"bytes"
	"fmt"
	"sort"

	"github.com/fatih/color"
	"github.com/sirupsen/logrus"
)

// ColorFormatter is a formatter that uses different colors for each level.
type ColorFormatter struct{}

var logColorMap = map[logrus.Level]*color.Color{
	logrus.PanicLevel: color.New(color.FgHiRed, color.Bold),
	logrus.FatalLevel: color.New(color.FgHiRed),
	logrus.ErrorLevel: color.New(color.FgRed),
	logrus.WarnLevel:  color.New(color.FgYellow),
	logrus.InfoLevel:  color.New(color.FgHiWhite),
	logrus.DebugLevel: color.New(color.FgCyan),
	logrus.TraceLevel: color.New(color.FgHiCyan),
}

func (f *ColorFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	buff := &bytes.Buffer{}
	if entry.Buffer != nil {
		buff = entry.Buffer
	}

	if color.NoColor {
		fmt.Fprintf(buff, "%s\n", entry.Message)
	} else {
		reset := color.New(color.Reset)
		reset.Fprintf(buff, "")
		logColor, ok := logColorMap[entry.Level]
		if !ok {
			panic(fmt.Sprintf("unexpected level: %v", entry.Level))
		}
		logColor.Fprintf(buff, "%s\n", entry.Message)
		reset.Fprintf(buff, "")
	}

	keys := []string{}
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintf(buff, "  %s=%v\n", k, entry.Data[k])
	}

	return buff.Bytes(), nil
}

func getLevel(logLevel string) (logrus.Level, error) {
	for _, level := range logrus.AllLevels {
		if logLevel == level.String() {
			return level, nil
		}
	}
	return logrus.TraceLevel, fmt.Errorf("invalid level %v", logLevel)
}

// Setup configures logrus standard logger with ColorFormatter and given log level.
func Setup(logLevel string) error {
	logrus.SetFormatter(&ColorFormatter{})

	level, err := getLevel(logLevel)
	if err != nil {
		return err
	}
	logrus.SetLevel(level)
	return nil
}
