package facesim

import (
	"context"
	"encoding/base64"
	"encoding/json"
	modelhubv2 "github.com/wgdl666/wgModelHub/gen/wg_model_hub/v2"
	"github.com/wgdl666/wgModelHub/models"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func pairRequest(images ...[]byte) *modelhubv2.GenerateRequest {
	parts := []*modelhubv2.ContentPart{}
	for _, b := range images {
		parts = append(parts, &modelhubv2.ContentPart{Content: &modelhubv2.ContentPart_Image{Image: &modelhubv2.Media{MimeType: "image/png", Source: &modelhubv2.Media_Data{Data: b}}}})
	}
	return &modelhubv2.GenerateRequest{Input: &modelhubv2.Input{Items: []*modelhubv2.InputItem{{Item: &modelhubv2.InputItem_Message{Message: &modelhubv2.Message{Parts: parts}}}}}}
}

func TestAnonymousG6CompareAndSignedCosine(t *testing.T) {
	s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/facesim/compare" || r.Header.Get("Authorization") != "" {
			t.Error("wrong route or unexpected authentication")
		}
		if err := r.ParseMultipartForm(1 << 20); err != nil {
			t.Fatal(err)
		}
		a, _ := base64.StdEncoding.DecodeString(r.FormValue("image_a"))
		b, _ := base64.StdEncoding.DecodeString(r.FormValue("image_b"))
		if string(a) != "original" || string(b) != "generated" {
			t.Fatal("image ordering changed")
		}
		io.WriteString(w, `{"success":true,"similarity":-0.25}`)
	}))
	defer s.Close()
	p, err := New("g6", s.URL+"/facesim", "", "")
	if err != nil {
		t.Fatal(err)
	}
	event, err := p.Generate(context.Background(), models.FaceSim, pairRequest([]byte("original"), []byte("generated")))
	if err != nil {
		t.Fatal(err)
	}
	var result map[string]float64
	if err := json.Unmarshal([]byte(event.GetItems()[0].GetText()), &result); err != nil {
		t.Fatal(err)
	}
	if result["similarity"] != 0 || result["cosine_similarity"] != -0.25 {
		t.Fatal("signed cosine must remain available")
	}
	if _, err := p.Generate(context.Background(), models.FaceSim, pairRequest([]byte("a"))); err == nil {
		t.Fatal("one image accepted")
	}
}
func TestNoFaceOrMissingScoreCannotBeSuccess(t *testing.T) {
	for _, body := range []string{`{"success":false,"similarity":null}`, `{"success":true}`, `{"success":true,"similarity":1.5}`} {
		t.Run(body, func(t *testing.T) {
			s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { io.WriteString(w, body) }))
			defer s.Close()
			p, _ := New("g6", s.URL, "", "")
			if _, err := p.Generate(context.Background(), models.FaceSim, pairRequest([]byte("a"), []byte("b"))); err == nil {
				t.Fatal("invalid face result accepted")
			}
		})
	}
	if _, err := New("g6", "http://example", "user", ""); err == nil {
		t.Fatal("partial credentials accepted")
	}
}
