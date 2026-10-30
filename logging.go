package main

import (
	"github.com/go-logr/logr"
	"go.uber.org/zap/zapcore"
	logsapi "k8s.io/component-base/logs/api/v1"
	"k8s.io/component-base/logs/json"
)

// jsonRFC3339Format is component-base's "json" format with RFC3339Nano timestamps.
const jsonRFC3339Format = "json-rfc3339"

// rfc3339JSONFactory overrides only the time encoder: component-base's json format hardcodes epoch-millis.
type rfc3339JSONFactory struct{}

func (rfc3339JSONFactory) Create(c logsapi.LoggingConfiguration, o logsapi.LoggingOptions) (logr.Logger, logsapi.RuntimeControl) {
	encoderConfig := &zapcore.EncoderConfig{
		MessageKey:     "msg",
		CallerKey:      "caller",
		NameKey:        "logger",
		TimeKey:        "ts",
		EncodeTime:     zapcore.RFC3339NanoTimeEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	// Single stream, as json.Factory without split-stream.
	stream := zapcore.Lock(json.AddNopSync(o.ErrorStream))
	return json.NewJSONLogger(c.Verbosity, stream, nil, encoderConfig)
}

// registerJSONRFC3339Format must run before flags are parsed: the format registry freezes when the flag set is built.
func registerJSONRFC3339Format() error {
	return logsapi.RegisterLogFormat(jsonRFC3339Format, rfc3339JSONFactory{}, logsapi.LoggingBetaOptions)
}
