package proxy

import (
	"time"

	"github.com/getsentry/sentry-go"
	"go.uber.org/zap"
)

// CaptureError sends to Sentry general exception info
// with sentry.Level and with some extra provided detail in params.
// Also write into log.
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
