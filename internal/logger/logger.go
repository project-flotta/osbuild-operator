package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger(logLevel string) (*zap.SugaredLogger, error) {
	var level zapcore.Level
	err := level.UnmarshalText([]byte(logLevel))
	if err != nil {
		return nil, err
	}
	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level.SetLevel(level)
	logBuilder, err := logConfig.Build()
	if err != nil {
		return nil, err
	}
	return logBuilder.Sugar(), nil
}
