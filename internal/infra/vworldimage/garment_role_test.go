package vworldimage

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"github.com/wgdl666/wgModelHub/models"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRegistrationForwardsExplicitRoleWithoutCrop(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if err := r.ParseMultipartForm(8 << 20); err != nil {
			t.Fatal(err)
		}
		if r.FormValue("garment_name") != "Black Embellished Sandals" || r.FormValue("garment_role") != "shoes" || r.FormValue("use_crop") != "false" {
			t.Fatalf("fields: %#v", r.Form)
		}
		if r.Form.Has("prompt") || r.Form.Has("parse_mask") || r.Form.Has("parse_labels") {
			t.Fatal("registration must retain server prompt and no parser crop")
		}
		json.NewEncoder(w).Encode(map[string]any{"data": []map[string]string{{"b64_json": base64.StdEncoding.EncodeToString(pngBytes(t))}}})
	}))
	defer server.Close()
	p := newTestProvider(t, server.URL)
	if _, err := p.GenerateImage(context.Background(), models.VWorldWardrobe10, wardrobeRequest(pngBytes(t), `{"garment_name":"Black Embellished Sandals","garment_role":"shoes"}`)); err != nil {
		t.Fatal(err)
	}
	if calls != 1 {
		t.Fatalf("generation calls=%d", calls)
	}
}

func TestInvalidOrConflictingRoleNeverReachesUpstream(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(500) }))
	defer server.Close()
	p := newTestProvider(t, server.URL)
	for _, meta := range []string{
		`{"garment_name":"shoe","garment_role":"feet"}`,
		`{"garment_name":"shoe","garment_role":"footwear.sandals"}`,
		`{"garment_name":"shoe","garment_role":"shoes","inventory_slot":"top"}`,
	} {
		if _, err := p.GenerateImage(context.Background(), models.VWorldWardrobe10, wardrobeRequest(pngBytes(t), meta)); err == nil {
			t.Fatalf("accepted %s", meta)
		}
	}
	// A role alone cannot authorize a parser crop when the slot is absent.
	if _, err := p.GenerateImage(context.Background(), models.VWorldWardrobe10, wardrobeRequest(pngBytes(t), `{"garment_name":"shoe","garment_role":"shoes","parse_labels":{"1":"feet"}}`, []byte("mask"))); err == nil {
		t.Fatal("mask must still require inventory_slot")
	}
	if calls != 0 {
		t.Fatalf("invalid requests reached upstream: %d", calls)
	}
}
