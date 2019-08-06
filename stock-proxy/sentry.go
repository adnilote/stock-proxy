package proxy

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

func NewLogger(outputPath []string) (*zap.Logger, error) {
	for _, path := range outputPath {
		if path != "stdout" {
			os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0666)
		}
	}

	cfg := zap.NewDevelopmentConfig()
	cfg.OutputPaths = outputPath
	return cfg.Build()
}

func ConfigureSentry(dsn string) {
	// dsn := config.GetSentryDSN()
	// if dsn == "" {
	// 	return
	// }

	err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		AttachStacktrace: true,
	})
	if err != nil {
		log.Printf("sentry initialization failed: %v", err)
	}
}

// CaptureException sends to Sentry general exception info with some extra provided detail (like user email, claim url etc)
func CaptureError(err error, level sentry.Level, lg *zap.Logger, params ...map[string]interface{}) {
	sentry.WithScope(func(scope *sentry.Scope) {
		var extra map[string]interface{}
		if len(params) > 0 {
			extra = params[0]
		} else {
			extra = map[string]interface{}{}
		}

		for k, v := range extra {
			scope.SetExtra(k, v)
		}

		scope.SetLevel(level)

		sentry.CaptureException(err)
	})

	if level == sentry.LevelFatal {
		lg.Fatal("", zap.Error(err))
	}
	if level == sentry.LevelError {
		lg.Error("", zap.Error(err))
	}
	sentry.Flush(time.Second * 5)
}
