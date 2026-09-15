// Copyright 2026 Baidu Inc. All rights reserved.
// Use of this source code is governed by a xxx
// license that can be found in the LICENSE file.

package log

import (
	"context"
	"io"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"icode.baidu.com/baidu/devops-ai/knowbase-server/pkg/trace"
)

type LoggerX struct {
	SugarLogger *zap.SugaredLogger
}

var Logger LoggerX

// fileWriter 日志文件写入器，供其他模块复用
var fileWriter *lumberjack.Logger

// GetWriter 获取日志文件的io.Writer，可用于ORM等组件的日志输出
func GetWriter() io.Writer {
	return fileWriter
}

// InitLogger 初始化日志
func InitLogger(level, filePath string, maxSize, maxBackups, maxAge int, compress bool) error {
	// 设置日志级别
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		zapLevel = zapcore.InfoLevel
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 文件输出
	fileWriter = &lumberjack.Logger{
		Filename:   filePath,
		MaxSize:    maxSize,
		MaxBackups: maxBackups,
		MaxAge:     maxAge,
		Compress:   compress,
	}

	// 同时输出到文件和控制台
	core := zapcore.NewTee(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(fileWriter),
			zapLevel,
		),
	)

	// 创建logger
	logger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	Logger = LoggerX{logger.Sugar()}
	defer func(SugarLogger *zap.SugaredLogger) {
		err := SugarLogger.Sync()
		if err != nil {
			Logger.Error(err, "logger sync failed")
		}
	}(Logger.SugarLogger)

	return nil
}

func (l LoggerX) Info(msg string, keysAndValues ...interface{}) {
	l.SugarLogger.Info(msg, keysAndValues)
}

func (l LoggerX) Error(err error, msg string, keysAndValues ...interface{}) {
	l.SugarLogger.Error(err, msg, keysAndValues)
}

func (l LoggerX) Infof(template string, args ...interface{}) {
	l.SugarLogger.Infof(template, args...)
}

func (l LoggerX) Errorf(template string, args ...interface{}) {
	l.SugarLogger.Errorf(template, args...)
}

func (l LoggerX) Warnf(template string, args ...interface{}) {
	l.SugarLogger.Warnf(template, args...)
}

func (l LoggerX) Debugf(template string, args ...interface{}) {
	l.SugarLogger.Debugf(template, args...)
}

func (l LoggerX) Fatalf(template string, args ...interface{}) {
	l.SugarLogger.Fatalf(template, args...)
}

// WithTrace 创建带TraceID的日志实例
// 返回新的LoggerX实例，所有日志都会自动携带traceId字段
func (l LoggerX) WithTrace(ctx context.Context) LoggerX {
	traceID := trace.FromContext(ctx)
	if traceID != "" {
		return LoggerX{
			SugarLogger: l.SugarLogger.With("traceId", traceID),
		}
	}
	return l
}

// InfofCtx 带context的Info日志
func (l LoggerX) InfofCtx(ctx context.Context, template string, args ...interface{}) {
	l.WithTrace(ctx).SugarLogger.Infof(template, args...)
}

// ErrorfCtx 带context的Error日志
func (l LoggerX) ErrorfCtx(ctx context.Context, template string, args ...interface{}) {
	l.WithTrace(ctx).SugarLogger.Errorf(template, args...)
}

// WarnfCtx 带context的Warn日志
func (l LoggerX) WarnfCtx(ctx context.Context, template string, args ...interface{}) {
	l.WithTrace(ctx).SugarLogger.Warnf(template, args...)
}

// DebugfCtx 带context的Debug日志
func (l LoggerX) DebugfCtx(ctx context.Context, template string, args ...interface{}) {
	l.WithTrace(ctx).SugarLogger.Debugf(template, args...)
}
