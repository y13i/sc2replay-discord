package main

import (
	"github.com/k0kubun/pp"
	"go.uber.org/zap"
)

type Logger struct {
	*zap.SugaredLogger
}

func (l Logger) Debug(args ...interface{}) {
	l.Debugf("", "\n"+pp.Sprint(args))
}

func getLogger(isProd bool) Logger {
	var (
		_logger *zap.Logger
		err     error
	)

	if isProd {
		_logger, err = zap.NewProduction()
	} else {
		_logger, err = zap.NewDevelopment()
	}

	if err != nil {
		panic(err)
	}

	return Logger{_logger.Sugar()}
}
