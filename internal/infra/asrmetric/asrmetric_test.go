package asrmetric_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/wgdl666/wgModelHub/internal/infra/asrmetric"
	"github.com/wgdl666/wgModelHub/internal/infra/metricserver"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

func TestSessionFailureCounterContract(t *testing.T) {
	reader, err := metricserver.NewReader()
	if err != nil {
		t.Fatalf("NewReader: %v", err)
	}
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(reader.ManualReader()))
	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })
	if err := asrmetric.Configure(provider.Meter("asrmetric-contract")); err != nil {
		t.Fatal(err)
	}
	asrmetric.Record(context.Background(), "assemblyai", "1008")

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
	if !strings.Contains(text, `wg_modelhub_asr_session_failures_total`) {
		t.Fatalf("missing counter\n%s", text)
	}
	if !strings.Contains(text, `provider="assemblyai"`) || !strings.Contains(text, `error_code="1008"`) {
		t.Fatalf("missing labels\n%s", text)
	}
	for _, bad := range []string{"user_id=", "session_id=", "task_id=", "error="} {
		if strings.Contains(text, bad) {
			t.Fatalf("forbidden label %s\n%s", bad, text)
		}
	}
}
