package logger

import (
	"log/syslog"
	"os"
	"path/filepath"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Options 日志相关配置
type Options struct {
	// 日志 syslog 的 tag（日志文件路径）
	Tag string
	// Debug 模式，debug 时，只把日志输出到 stdout
	Debug bool
}

// New constructs a new Logger
func New(options *Options) (*zap.Logger, error) {
	encoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())

	// 开启 Debug 时，只开启 stdout 日志输出
	if options.Debug {
		return zap.New(zapcore.NewCore(encoder, os.Stdout, zap.DebugLevel)), nil
	}

	tag := filepath.Clean(strings.ReplaceAll(strings.Trim(options.Tag, " /"), ".", "/"))
	writer, err := syslog.New(syslog.LOG_LOCAL0|syslog.LOG_INFO, tag)
	if err != nil {
		return nil, err
	}
	var cores []zapcore.Core
	cores = append(cores, NewCore(zapcore.InfoLevel, encoder, writer))

	return zap.New(zapcore.NewTee(cores...)), nil
}
