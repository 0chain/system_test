package util

import (
	"os"
	"strings"

	"github.com/sirupsen/logrus"
)

type FallbackLogger struct {
	logger logrus.Logger
}

func (l *FallbackLogger) Init() {
	logger := logrus.New()
	logger.Out = os.Stdout

	logger.SetFormatter(&logrus.TextFormatter{
		DisableQuote: true,
	})

	if strings.EqualFold(strings.TrimSpace(os.Getenv("DEBUG")), "true") {
		logger.SetLevel(logrus.DebugLevel)
	}
}

func (l *FallbackLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}
