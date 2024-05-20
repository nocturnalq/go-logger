package gologger

import "github.com/rs/zerolog"

type LoggerConfig struct {
	SentryDSN       string `yaml:"sentry_dsn" env:"SENTRY_DSN"`
	SentryDebugMode bool   `yaml:"sentry_debug_mode" env:"SENTRY_DEBUG_MODE" env-default:"false"`
	LogLevel        string `yaml:"log_level" env:"LOG_LEVEL"`
}

var LogLevelMapping map[string]zerolog.Level = map[string]zerolog.Level{
	"debug":    zerolog.DebugLevel,
	"info":     zerolog.InfoLevel,
	"warn":     zerolog.WarnLevel,
	"error":    zerolog.ErrorLevel,
	"fatal":    zerolog.FatalLevel,
	"panic":    zerolog.PanicLevel,
	"nolevel":  zerolog.NoLevel,
	"disabled": zerolog.Disabled,
	"trace":    zerolog.TraceLevel,
}
