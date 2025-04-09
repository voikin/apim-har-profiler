package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"github.com/rs/zerolog"
)

type Config struct {
	Level  string
	Pretty bool
}

var (
	Logger zerolog.Logger //nolint:gochecknoglobals // global by design
)

func New(cfg *Config) zerolog.Logger {
	var output io.Writer = os.Stdout
	if cfg.Pretty {
		output = zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: time.RFC3339,
		}
	}

	level, err := zerolog.ParseLevel(cfg.Level)
	if err != nil {
		level = zerolog.InfoLevel
	}

	zerolog.SetGlobalLevel(level)

	return zerolog.New(output).With().Timestamp().Logger()
}

func InitGlobalLogger(cfg *Config) {
	Logger = New(cfg)
	Logger.Info().Str("logger_level", Logger.GetLevel().String()).Msg("logger initialized")
}

func InterceptorLogger(l zerolog.Logger) logging.Logger {
	return logging.LoggerFunc(func(_ context.Context, lvl logging.Level, msg string, fields ...any) {
		interceptorLogger := l.With().Fields(fields).Logger()

		switch lvl {
		case logging.LevelDebug:
			interceptorLogger.Debug().Msg(msg)
		case logging.LevelInfo:
			interceptorLogger.Info().Msg(msg)
		case logging.LevelWarn:
			interceptorLogger.Warn().Msg(msg)
		case logging.LevelError:
			interceptorLogger.Error().Msg(msg)
		default:
			panic(fmt.Sprintf("unknown level %v", lvl))
		}
	})
}
