package log

import (
	"context"

	"github.com/sirupsen/logrus"
)

type loggerKeyType string

var loggerKey = loggerKeyType("logger")

func SetLoggerValue(ctx context.Context, logLevelStr string) context.Context {
	logger := logrus.New()

	logger.SetFormatter(&colorFormatter{})

	var level *logrus.Level
	for _, l := range logrus.AllLevels {
		if logLevelStr == l.String() {
			level = &l
		}
	}
	if level == nil {
		logger.Fatalf("invalid log level %v", logLevelStr)
	}
	logger.SetLevel(*level)

	return context.WithValue(ctx, loggerKey, logger)
}

func GetLogger(ctx context.Context) *logrus.Logger {
	logger, ok := ctx.Value(loggerKey).(*logrus.Logger)
	if !ok {
		panic("logger value missing from context")
	}
	return logger
}
