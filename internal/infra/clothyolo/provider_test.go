package clothyolo

import (
	"context"
	modelhubv2 "github.com/wgdl666/wgModelHub/gen/wg_model_hub/v2"
	"github.com/wgdl666/wgModelHub/models"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func clothRequest(images ...[]byte) *modelhubv2.GenerateRequest {
	parts := []*modelhubv2.ContentPart{}
	for _, b := range images {
		parts = append(parts, &modelhubv2.ContentPart{Content: &modelhubv2.ContentPart_Image{Image: &modelhubv2.Media{MimeType: "image/png", Source: &modelhubv2.Media_Data{Data: b}}}})
	}
	return &modelhubv2.GenerateRequest{Input: &modelhubv2.Input{Items: []*modelhubv2.InputItem{{Item: &modelhubv2.InputItem_Message{Message: &modelhubv2.Message{Parts: parts}}}}}}
}

func TestAnonymousG6PrefixAndClothingResult(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/cloth-yolo/predict" || r.Header.Get("Authorization") != "" {
			t.Errorf("wrong route or unexpected authentication")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		f, _, err := r.FormFile("file")
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		b, _ := io.ReadAll(f)
		if string(b) != "image bytes" || r.FormValue("conf") != "0.25" || r.FormValue("iou") != "0.45" {
			t.Error("multipart contract changed")
		}
		io.WriteString(w, `{"success":true,"items":[{"class_name":"shirt","bbox":[1,2,30,40]}]}`)
	}))
	defer s.Close()
	p, err := New("g6", s.URL+"/cloth-yolo", "", "")
	if err != nil {
		t.Fatal(err)
	}
	event, err := p.Generate(context.Background(), models.ClothYOLO, clothRequest([]byte("image bytes")))
	if err != nil {
		t.Fatal(err)
	}
	if event.GetItems()[0].GetText() != `{"success":true,"items":[{"class_name":"shirt","bbox":[1,2,30,40]}]}` {
		t.Fatal("clothing result was altered")
	}
	if _, err := p.Generate(context.Background(), models.ClothYOLO, clothRequest([]byte("a"), []byte("b"))); err == nil {
		t.Fatal("two images must be rejected")
	}
}
func TestLogicalFailureAndInvalidJSONAreRejected(t *testing.T) {
	for _, body := range []string{`{"success":false,"error":"failed"}`, `not json`} {
		t.Run(body, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
			defer s.Close()
			p, _ := New("g6", s.URL, "", "")
			if _, err := p.Generate(context.Background(), models.ClothYOLO, clothRequest([]byte("a"))); err == nil {
				t.Fatal("invalid success accepted")
			}
		})
	}
	if _, err := New("g6", "http://example", "user", ""); err == nil {
		t.Fatal("partial credentials accepted")
	}
}
