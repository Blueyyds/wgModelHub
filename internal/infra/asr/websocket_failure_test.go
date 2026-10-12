package asr

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gorilla/websocket"
	"github.com/wgdl666/wgModelHub/internal/infra/asrmetric"
	"github.com/wgdl666/wgModelHub/internal/infra/metricserver"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestErrorFrameAndCloseCountOnce(t *testing.T) {
	reader, err := metricserver.NewReader()
	if err != nil {
		t.Fatal(err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader.ManualReader()))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	if err := asrmetric.Configure(provider.Meter("asr-once")); err != nil {
		t.Fatal(err)
	}
	session := &webSocketSession{name: "assemblyai", ctx: context.Background()}
	counted := false
	session.noteSessionFailure(&SessionFailure{Code: "1008", Reason: "balance"}, &counted)
	session.noteSessionFailure(&websocket.CloseError{Code: websocket.ClosePolicyViolation, Text: "See Error message for details"}, &counted)

	ts := httptest.NewServer(reader.Handler())
	t.Cleanup(ts.Close)
	resp, err := http.Get(ts.URL)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	if !strings.Contains(text, `wg_modelhub_asr_session_failures_total{error_code="1008",provider="assemblyai"} 1`) &&
		!strings.Contains(text, `wg_modelhub_asr_session_failures_total{provider="assemblyai",error_code="1008"} 1`) {
		t.Fatalf("expected one sample\n%s", text)
	}
}
