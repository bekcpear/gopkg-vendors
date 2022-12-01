package logger

import (
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type level string

const (
	LevelDebug level = "debug"
	LevelInfo  level = "info"
	LevelWarn  level = "warn"
	LevelError level = "error"
	LevelPanic level = "panic"
)

type encoder string

const (
	EncoderConsole encoder = "console"
	EncoderJSON    encoder = "json"
)

type timeEncoding string

const (
	TimeEncodingISO8601 timeEncoding = "iso8601"
	TimeEncodingMillis  timeEncoding = "millis"
	TimeEncodingNanos   timeEncoding = "nano"
	TimeEncodingEpoch   timeEncoding = "epoch"
	TimeEncodingDefault timeEncoding = "default"
)

type ZapConfig struct {
	Development  bool         `json:"development,omitempty" yaml:"development,omitempty"`
	Encoder      encoder      `json:"encoder,omitempty" yaml:"encoder,omitempty"`
	Level        level        `json:"level,omitempty" yaml:"level,omitempty"`
	StackLevel   level        `json:"stack_level,omitempty" yaml:"stack_level,omitempty"`
	Sample       bool         `json:"sample,omitempty" yaml:"sample,omitempty"`
	TimeEncoding timeEncoding `json:"time_encoding,omitempty" yaml:"time_encoding,omitempty"`
}

type zapConfig struct {
	level      zap.AtomicLevel
	stackLevel zapcore.Level
	encoder    zapcore.Encoder
	opts       []zap.Option
	sample     bool
}

type encoderConfigFunc func(*zapcore.EncoderConfig)
type encoderFunc func(...encoderConfigFunc) zapcore.Encoder

func NewConfig(c *ZapConfig) (*zapConfig, error) {
	var zc zapConfig
	var eFunc encoderFunc

	// If development is enabled, use the default development config;
	// otherwise, use the default production config.
	if c.Development {
		eFunc, _ = getEncoder(EncoderConsole)
		zc.level = zap.NewAtomicLevelAt(zap.DebugLevel)
		zc.opts = append(zc.opts, zap.Development())
		zc.sample = false
		zc.stackLevel = zap.WarnLevel
	} else {
		eFunc, _ = getEncoder(EncoderJSON)
		zc.level = zap.NewAtomicLevelAt(zap.InfoLevel)
		zc.sample = true
	}

	// If Level is set, override the default Level
	if c.Level != "" {
		lvl, err := getLevel(c.Level)
		if err != nil {
			return nil, err
		}
		zc.level = zap.NewAtomicLevelAt(lvl)
	}

	// If StackLevel is set, override the default StackLevel
	if c.StackLevel != "" {
		lvl, err := getLevel(c.StackLevel)
		if err != nil {
			return nil, err
		}
		zc.stackLevel = lvl
	}
	zc.opts = append(zc.opts, zap.AddStacktrace(zc.stackLevel))

	// If Encoder is set, override the default Encoder
	if c.Encoder != "" {
		f, err := getEncoder(c.Encoder)
		if err != nil {
			return nil, err
		}
		eFunc = f
	}

	// Set TimeEncoding, use "2006-01-02 15:04:05" by default
	var ecFuncs []encoderConfigFunc
	if c.TimeEncoding != "" {
		tec, err := getTimeEncoder(c.TimeEncoding)
		if err != nil {
			return nil, err
		}
		ecFuncs = append(ecFuncs, withTimeEncoding(tec))
	} else {
		tec, _ := getTimeEncoder(TimeEncodingDefault)
		ecFuncs = append(ecFuncs, withTimeEncoding(tec))
	}
	zc.encoder = eFunc(ecFuncs...)

	zc.sample = c.Sample
	if zc.level.Level() < -1 {
		zc.sample = false
	}

	return &zc, nil
}

func getLevel(l level) (zapcore.Level, error) {
	var lvl zapcore.Level
	switch l {
	case LevelDebug:
		lvl = zapcore.DebugLevel
	case LevelInfo:
		lvl = zapcore.InfoLevel
	case LevelWarn:
		lvl = zapcore.WarnLevel
	case LevelError:
		lvl = zapcore.ErrorLevel
	case LevelPanic:
		lvl = zapcore.PanicLevel
	default:
		return lvl, fmt.Errorf("invalid log level \"%s\"", l)
	}
	return lvl, nil
}

func getEncoder(ec encoder) (encoderFunc, error) {
	switch ec {
	case EncoderConsole:
		return func(ecfs ...encoderConfigFunc) zapcore.Encoder {
			encoderConfig := zap.NewDevelopmentEncoderConfig()
			for _, f := range ecfs {
				f(&encoderConfig)
			}
			encoderConfig.ConsoleSeparator = "    "
			return zapcore.NewConsoleEncoder(encoderConfig)
		}, nil
	case EncoderJSON:
		return func(ecfs ...encoderConfigFunc) zapcore.Encoder {
			encoderConfig := zap.NewProductionEncoderConfig()
			for _, f := range ecfs {
				f(&encoderConfig)
			}
			return zapcore.NewJSONEncoder(encoderConfig)
		}, nil
	default:
		return nil, fmt.Errorf("invalid encoder \"%s\"", ec)
	}
}

func getTimeEncoder(tec timeEncoding) (zapcore.TimeEncoder, error) {
	switch tec {
	case TimeEncodingISO8601:
		return zapcore.ISO8601TimeEncoder, nil
	case TimeEncodingMillis:
		return zapcore.EpochMillisTimeEncoder, nil
	case TimeEncodingNanos:
		return zapcore.EpochNanosTimeEncoder, nil
	case TimeEncodingEpoch:
		return zapcore.EpochTimeEncoder, nil
	case TimeEncodingDefault:
		return func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
			enc.AppendString(t.Format("2006-01-02 15:04:05"))
		}, nil
	default:
		return nil, fmt.Errorf("invalid time encoder \"%s\"", tec)
	}
}

func withTimeEncoding(tec zapcore.TimeEncoder) encoderConfigFunc {
	return func(ec *zapcore.EncoderConfig) {
		ec.EncodeTime = tec
	}
}
