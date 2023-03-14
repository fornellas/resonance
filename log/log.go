package log

import (
	"context"
	"io"

	"github.com/sirupsen/logrus"
)

type loggerKeyType string

var loggerKey = loggerKeyType("logger")

// SetLoggerValue returns a copy of the context with a logger value set.
func SetLoggerValue(
	ctx context.Context, output io.Writer, logLevelStr string, exitFunc func(int),
) context.Context {
	logger := logrus.New()

	logger.SetOutput(output)
	logger.SetFormatter(&ColorFormatter{})
	logger.ExitFunc = exitFunc

	var level *logrus.Level
	for _, l := range logrus.AllLevels {
		if logLevelStr == l.String() {
			level = &l
			break
		}
	}
	if level == nil {
		logger.Fatalf("invalid log level %v", logLevelStr)
	}
	logger.SetLevel(*level)

	return context.WithValue(ctx, loggerKey, logger)
}

// GetLogger returns a logger previously set with SetLoggerValue.
func GetLogger(ctx context.Context) *logrus.Logger {
	logger, ok := ctx.Value(loggerKey).(*logrus.Logger)
	if !ok {
		panic("logger value missing from context")
	}
	return logger
}

// IndentLogger receives a context with a previously set logger (SetLoggerValue),
// and returns a copy of it but with a logger with one extra indentation.
func IndentLogger(ctx context.Context) context.Context {
	oldLogger := GetLogger(ctx)
	newLogger := logrus.New()
	newLogger.SetOutput(oldLogger.Out)
	newLogger.SetFormatter(&ColorFormatter{
		Indent: oldLogger.Formatter.(*ColorFormatter).Indent + 1,
	})
	newLogger.ExitFunc = oldLogger.ExitFunc
	newLogger.SetLevel(oldLogger.Level)
	return context.WithValue(ctx, loggerKey, newLogger)
}
