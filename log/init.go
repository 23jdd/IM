package log

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Logger 全局日志实例，初始化后供包内各日志函数使用
var Logger *zap.Logger

// InitLogger 初始化全局 zap 日志，按 level 设置日志级别，同时输出到滚动日志文件和控制台
func InitLogger(logPath string, level string) {
	lvl := zapcore.InfoLevel
	switch level {
	case "debug":
		lvl = zapcore.DebugLevel
	case "warn":
		lvl = zapcore.WarnLevel
	case "error":
		lvl = zapcore.ErrorLevel
	}

	// 日志文件按大小滚动切割，并按数量和天数保留历史，开启压缩
	fileWriter := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	}

	consoleWriter := zapcore.Lock(os.Stdout)

	encoder := zapcore.NewJSONEncoder(zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		EncodeTime:    zapcore.ISO8601TimeEncoder,
		EncodeLevel:   zapcore.LowercaseLevelEncoder,
		EncodeCaller:  zapcore.ShortCallerEncoder,
	})

	// 使用 Tee 将日志同时写入文件和控制台两个 Core
	core := zapcore.NewTee(
		zapcore.NewCore(encoder, zapcore.AddSync(fileWriter), lvl),
		zapcore.NewCore(encoder, consoleWriter, lvl),
	)

	Logger = zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}

// CloseLogger 关闭日志，刷新缓冲区中未写入的日志
func CloseLogger() {
	if Logger != nil {
		Logger.Sync()
	}
}

// Info 记录 Info 级别日志
func Info(msg string, fields ...zap.Field) {
	Logger.Info(msg, fields...)
}

// Error 记录 Error 级别日志
func Error(msg string, fields ...zap.Field) {
	Logger.Error(msg, fields...)
}

// Warn 记录 Warn 级别日志
func Warn(msg string, fields ...zap.Field) {
	Logger.Warn(msg, fields...)
}

// Debug 记录 Debug 级别日志
func Debug(msg string, fields ...zap.Field) {
	Logger.Debug(msg, fields...)
}
