package logger

import (
	"errors"
	"io"

	"go.uber.org/zap"
)

const maxLogLen = 1024 * 10

type logWriter struct {
	level level
	l     *zap.SugaredLogger
}

func (lw *logWriter) Write(p []byte) (n int, err error) {
	switch lw.level {
	case LevelDebug:
		if len(p) > maxLogLen {
			lw.l.Debugw(string(p[:maxLogLen]), "truncated", true)
			return maxLogLen, errors.New("log line too long(max 10240)")
		} else {
			lw.l.Debug(string(p))
			return len(p), nil
		}
	case LevelInfo:
		if len(p) > maxLogLen {
			lw.l.Infow(string(p[:maxLogLen]), "truncated", true)
			return maxLogLen, errors.New("log line too long(max 10240)")
		} else {
			lw.l.Info(string(p))
			return len(p), nil
		}
	case LevelWarn:
		if len(p) > maxLogLen {
			lw.l.Warnw(string(p[:maxLogLen]), "truncated", true)
			return maxLogLen, errors.New("log line too long(max 10240)")
		} else {
			lw.l.Warn(string(p))
			return len(p), nil
		}
	case LevelError:
		if len(p) > maxLogLen {
			lw.l.Errorw(string(p[:maxLogLen]), "truncated", true)
			return maxLogLen, errors.New("log line too long(max 10240)")
		} else {
			lw.l.Error(string(p))
			return len(p), nil
		}
	case LevelPanic:
		if len(p) > maxLogLen {
			lw.l.Panicw(string(p[:maxLogLen]), "truncated", true)
			return maxLogLen, errors.New("log line too long(max 10240)")
		} else {
			lw.l.Panic(string(p))
			return len(p), nil
		}
	default:
		if len(p) > maxLogLen {
			lw.l.Infow(string(p[:maxLogLen]), "truncated", true)
			return maxLogLen, errors.New("log line too long(max 10240)")
		} else {
			lw.l.Info(string(p))
			return len(p), nil
		}
	}

}

func NewLogWriter() io.Writer {
	return NewLogWriterWithLevel(LevelInfo)
}

func NewLogWriterWithLevel(lv level) io.Writer {
	return &logWriter{
		level: lv,
		l:     logger,
	}
}
