package metrics_otel

import (
	"go.opentelemetry.io/otel/attribute"
)

var AttrPhase = attribute.Key("phase")
var AttrResult = attribute.Key("result")
var AttrStatusClass = attribute.Key("status_class")
var AttrVerb = attribute.Key("verb")
var AttrResourceKind = attribute.Key("resource_kind")
var AttrForced = attribute.Key("forced")
var AttrReason = attribute.Key("reason")
var AttrOperation = attribute.Key("operation")
var AttrSection = attribute.Key("section")
var AttrOutcome = attribute.Key("outcome")

func StatusClass(code int) string {
	if code == 0 {
		return "0"
	}
	switch {
	case code >= 100 && code < 200:
		return "1xx"
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "unknown"
	}
}
