package logger

import (
	"context"
	"os"
	"strings"
	"time"

	"github.com/Keneke-Einar/delivertrack/pkg/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

type contextKey string

const correlationIDKey contextKey = "correlation_id"

// Logger wraps zap.Logger
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new structured logger based on configuration
func NewLogger(cfg config.LoggingConfig, serviceName string) (*Logger, error) {
	var encoder zapcore.Encoder
	encoderConfig := zap.NewProductionEncoderConfig()

	if cfg.Format == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	var writeSyncer zapcore.WriteSyncer
	if cfg.Output == "stdout" {
		writeSyncer = zapcore.AddSync(os.Stdout)
	} else {
		// Use lumberjack for file rotation
		writeSyncer = zapcore.AddSync(&lumberjack.Logger{
			Filename:   cfg.Output,
			MaxSize:    cfg.Rotation.MaxSize,
			MaxBackups: cfg.Rotation.MaxBackups,
			MaxAge:     cfg.Rotation.MaxAge,
			Compress:   cfg.Rotation.Compress,
		})
	}

	level, err := zapcore.ParseLevel(strings.ToLower(cfg.Level))
	if err != nil {
		level = zapcore.InfoLevel
	}

	core := zapcore.NewCore(encoder, writeSyncer, level)

	// Add sampling
	core = zapcore.NewSamplerWithOptions(core, 100*time.Millisecond, cfg.Sampling.Initial, cfg.Sampling.Thereafter)

	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	// Add service name as a field
	zapLogger = zapLogger.With(zap.String("service", serviceName))

	return &Logger{Logger: zapLogger}, nil
}

// WithContext adds correlation ID from context
func (l *Logger) WithContext(ctx context.Context) *Logger {
	if correlationID, ok := ctx.Value(correlationIDKey).(string); ok {
		return &Logger{Logger: l.Logger.With(zap.String("correlation_id", correlationID))}
	}
	return l
}

// WithFields adds structured fields
func (l *Logger) WithFields(fields ...zap.Field) *Logger {
	return &Logger{Logger: l.Logger.With(fields...)}
}

// InfoWithFields logs an info message with context and fields
func (l *Logger) InfoWithFields(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).With(fields...).Info(msg)
}

// ErrorWithFields logs an error message with context and fields
func (l *Logger) ErrorWithFields(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).With(fields...).Error(msg)
}

// WarnWithFields logs a warning message with context and fields
func (l *Logger) WarnWithFields(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).With(fields...).Warn(msg)
}

// DebugWithFields logs a debug message with context and fields
func (l *Logger) DebugWithFields(ctx context.Context, msg string, fields ...zap.Field) {
	l.WithContext(ctx).With(fields...).Debug(msg)
}

// SetCorrelationID sets correlation ID in context
func SetCorrelationID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, correlationIDKey, id)
}

// GetCorrelationID gets correlation ID from context
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(correlationIDKey).(string); ok {
		return id
	}
	return ""
}