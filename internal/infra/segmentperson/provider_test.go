package segmentperson

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"testing"

	modelhubv2 "github.com/wgdl666/wgModelHub/gen/wg_model_hub/v2"
	"github.com/wgdl666/wgModelHub/internal/provider"
	"github.com/wgdl666/wgModelHub/models"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func pngBytes(t *testing.T) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{R: 255, A: 128})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestPersonEndpointSendsExistingFormContract(t *testing.T) {
	cutout := pngBytes(t)
	var gotAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/segment" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("method") != "person-bria-rmbg" || r.FormValue("output") != "b64" || r.FormValue("preprocess") != "false" {
			t.Fatalf("form = %#v", r.Form)
		}
		if r.FormValue("image") != base64.StdEncoding.EncodeToString(cutout) {
			t.Fatal("image field is not the inline input")
		}
		if _, ok := r.Form["model"]; ok {
			t.Fatal("model id must not be sent upstream")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"refined_png_b64": base64.StdEncoding.EncodeToString(cutout)})
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL, "user", "secret")
	event, err := p.GenerateImage(context.Background(), models.SegmentPersonBria, imageRequest(inlinePNG(cutout)))
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth == "" {
		t.Fatal("basic auth was not sent")
	}
	got := event.GetItems()[0].GetImage().GetData()
	if string(got) != string(cutout) {
		t.Fatalf("png len = %d", len(got))
	}
}

func TestG6UsesPortraitForPersonAndBriaForSubject(t *testing.T) {
	cutout := pngBytes(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Fatal(err)
		}
		expected := "person-bria-rmbg"
		if r.URL.Path == "/segment-person/api/segment" {
			expected = "person-birefnet-portrait"
		} else if r.URL.Path != "/segment-person/api/segment_subject" {
			t.Fatal("wrong G6 path")
		}
		if r.FormValue("method") != expected || r.Header.Get("Authorization") != "" {
			t.Fatal("wrong endpoint model or authentication")
		}
		json.NewEncoder(w).Encode(map[string]string{"refined_png_b64": base64.StdEncoding.EncodeToString(cutout)})
	}))
	defer server.Close()
	p, err := NewWithMethods("g6", server.URL+"/segment-person", "", "", "person-bria-rmbg", "person-birefnet-portrait", "person-bria-rmbg")
	if err != nil {
		t.Fatal(err)
	}
	for _, model := range []string{models.SegmentPersonBria, models.SegmentSubjectBria} {
		if _, err := p.GenerateImage(context.Background(), model, imageRequest(inlinePNG(cutout))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSubjectEndpointOmitsPreprocess(t *testing.T) {
	cutout := pngBytes(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/segment_subject" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Fatal(err)
		}
		if r.Form.Has("preprocess") || r.Header.Get("Authorization") != "" {
			t.Fatalf("form=%#v auth=%q", r.Form, r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"refined_png_b64": base64.StdEncoding.EncodeToString(cutout)})
	}))
	defer server.Close()

	p := newTestProvider(t, server.URL, "", "")
	if _, err := p.GenerateImage(context.Background(), models.SegmentSubjectBria, imageRequest(inlinePNG(cutout))); err != nil {
		t.Fatal(err)
	}
}

func TestSegmentSendsTraceIDOnlyWhenSpanIsValid(t *testing.T) {
	cutout := pngBytes(t)
	var traces []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(4 << 20); err != nil {
			t.Fatal(err)
		}
		if r.Form.Has("trace_id") {
			traces = append(traces, r.FormValue("trace_id"))
		} else {
			traces = append(traces, "")
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"refined_png_b64": base64.StdEncoding.EncodeToString(cutout)})
	}))
	defer server.Close()
	p := newTestProvider(t, server.URL, "user", "secret")
	req := imageRequest(inlinePNG(cutout))
	if _, err := p.GenerateImage(context.Background(), models.SegmentPersonBria, req); err != nil {
		t.Fatal(err)
	}
	tp := sdktrace.NewTracerProvider()
	ctx, span := tp.Tracer("test").Start(context.Background(), "modelhub.Generate")
	defer span.End()
	if _, err := p.GenerateImage(ctx, models.SegmentSubjectBria, req); err != nil {
		t.Fatal(err)
	}
	want := span.SpanContext().TraceID().String()
	if len(traces) != 2 || traces[0] != "" || traces[1] != want {
		t.Fatalf("traces=%v want [\"\" %s]", traces, want)
	}
}

func TestRejectsWrongImageCountAndUnknownModel(t *testing.T) {
	pngData := pngBytes(t)
	p := newTestProvider(t, "https://segment.example", "", "")
	if _, err := p.GenerateImage(context.Background(), models.SegmentPersonBria, imageRequest()); err == nil {
		t.Fatal("expected missing image error")
	}
	if _, err := p.GenerateImage(context.Background(), models.SegmentPersonBria, imageRequest(inlinePNG(pngData), inlinePNG(pngData))); err == nil {
		t.Fatal("expected extra image error")
	}
	if _, err := p.GenerateImage(context.Background(), models.PhotoroomSegment, imageRequest(inlinePNG(pngData))); err == nil {
		t.Fatal("expected unknown model error")
	}
}

func TestNewRejectsHalfAuth(t *testing.T) {
	_, err := New("segment", "https://segment.example", "user", "", "person-bria-rmbg")
	var providerErr *provider.Error
	if !errors.As(err, &providerErr) || providerErr.Kind != provider.ErrorConfiguration {
		t.Fatalf("err = %v", err)
	}
}

func imageRequest(images ...*modelhubv2.Media) *modelhubv2.GenerateRequest {
	parts := make([]*modelhubv2.ContentPart, 0, len(images))
	for _, image := range images {
		parts = append(parts, &modelhubv2.ContentPart{Content: &modelhubv2.ContentPart_Image{Image: image}})
	}
	return &modelhubv2.GenerateRequest{
		Input: &modelhubv2.Input{Items: []*modelhubv2.InputItem{{
			Item: &modelhubv2.InputItem_Message{Message: &modelhubv2.Message{
				Role:  modelhubv2.Role_ROLE_USER,
				Parts: parts,
			}},
		}}},
	}
}

func inlinePNG(data []byte) *modelhubv2.Media {
	return &modelhubv2.Media{MimeType: "image/png", Source: &modelhubv2.Media_Data{Data: data}}
}

func newTestProvider(t *testing.T, baseURL, username, password string) *Provider {
	t.Helper()
	p, err := New("segment_person_bria", baseURL, username, password, "person-bria-rmbg")
	if err != nil {
		t.Fatal(err)
	}
	return p
}
