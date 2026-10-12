package asrmetric

import (
	"context"
	"fmt"
	"sync/atomic"

	"go.opentelemetry.io/otel/attribute"
	otelmetric "go.opentelemetry.io/otel/metric"
)

type recorder struct {
	failures otelmetric.Int64Counter
}

var current atomic.Pointer[recorder]

// Configure 把 ASR 会话拒绝注册到 Prometheus Meter。不向 Logfire 导出，避免和抓取指标双写。
func Configure(meter otelmetric.Meter) error {
	failures, err := meter.Int64Counter(
		"wg.modelhub.asr.session_failures",
		otelmetric.WithDescription("ASR provider rejected a new streaming session"),
	)
	if err != nil {
		return fmt.Errorf("create wg.modelhub.asr.session_failures: %w", err)
	}
	current.Store(&recorder{failures: failures})
	return nil
}

// Record 记一次新会话被供应商拒绝。标签只有 provider 与数字 error_code。
func Record(ctx context.Context, provider, errorCode string) {
	r := current.Load()
	if r == nil || provider == "" || errorCode == "" {
		return
	}
	r.failures.Add(ctx, 1, otelmetric.WithAttributes(
		attribute.String("provider", provider),
		attribute.String("error_code", errorCode),
	))
}
