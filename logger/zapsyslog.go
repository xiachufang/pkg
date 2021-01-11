package logger

import (
	"log/syslog"

	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

// Core zap syslog Core struct
type Core struct {
	zapcore.LevelEnabler
	encoder zapcore.Encoder
	writer  *syslog.Writer
}

// NewCore return a new syslog zap Core
// nolint: gocritic
func NewCore(level zapcore.LevelEnabler, encoder zapcore.Encoder, writer *syslog.Writer) *Core {
	return &Core{
		LevelEnabler: level,
		encoder:      encoder,
		writer:       writer,
	}
}

// With adds structured context to the Core.
func (core *Core) With(fields []zapcore.Field) zapcore.Core {
	clone := core.clone()
	for _, field := range fields {
		field.AddTo(clone.encoder)
	}
	return clone
}

// Check determines whether the supplied Entry should be logged
// nolint: gocritic
func (core *Core) Check(entry zapcore.Entry, checked *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if core.Enabled(entry.Level) {
		return checked.AddCore(entry, core)
	}
	return checked
}

// Write serializes the Entry and any Fields supplied at the log site and
// writes them to their destination.
// nolint: gocritic
func (core *Core) Write(entry zapcore.Entry, fields []zapcore.Field) error {
	// Generate the message.
	buffer, err := core.encoder.EncodeEntry(entry, fields)
	if err != nil {
		return errors.Wrap(err, "failed to encode log entry")
	}
	defer buffer.Free()

	message := buffer.String()

	// Write the message.
	switch entry.Level {
	case zapcore.DebugLevel:
		return core.writer.Debug(message)

	case zapcore.InfoLevel:
		return core.writer.Info(message)

	case zapcore.WarnLevel:
		return core.writer.Warning(message)

	case zapcore.ErrorLevel:
		return core.writer.Err(message)

	case zapcore.DPanicLevel:
		return core.writer.Crit(message)

	case zapcore.PanicLevel:
		return core.writer.Crit(message)

	case zapcore.FatalLevel:
		return core.writer.Crit(message)

	default:
		return errors.Errorf("unknown log level: %v", entry.Level)
	}
}

// Sync flushes buffered logs (if any).
func (core *Core) Sync() error {
	return nil
}

func (core *Core) clone() *Core {
	return &Core{
		LevelEnabler: core.LevelEnabler,
		encoder:      core.encoder.Clone(),
		writer:       core.writer,
	}
}
