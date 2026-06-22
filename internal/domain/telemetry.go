package domain

import "fmt"

const (
	MaxTelemetryBodyBytes   = 256 * 1024
	MaxTelemetryBatchEvents = 500
)

var ErrTelemetryPayloadTooLarge = fmt.Errorf(
	"telemetry payload exceeds maximum size of %dKB or %d events per array",
	MaxTelemetryBodyBytes/1024, MaxTelemetryBatchEvents,
)
