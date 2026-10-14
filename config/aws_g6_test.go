package config

import (
	"testing"

	"github.com/wgdl666/wgModelHub/models"
)

func TestG6AnonymousProvidersAndEndpointMethods(t *testing.T) {
	c := validConfig()
	c.Providers["g6-cloth"] = ProviderConfig{Models: []string{models.ClothYOLO}, ClothYOLO: &ClothYOLOProviderConfig{BaseURL: "http://172.31.43.7:18280/cloth-yolo"}}
	c.Providers["g6-face"] = ProviderConfig{Models: []string{models.FaceSim}, FaceSim: &FaceSimProviderConfig{BaseURL: "http://172.31.43.7:18280/facesim"}}
	c.Providers["g6-human"] = ProviderConfig{Models: []string{models.HumanYOLO}, HumanYOLO: &HumanYOLOProviderConfig{BaseURL: "http://172.31.43.7:18280/human-yolo"}}
	c.Providers["g6-segment"] = ProviderConfig{Models: []string{models.SegmentPersonBria, models.SegmentSubjectBria}, SegmentPerson: &SegmentPersonProviderConfig{BaseURL: "http://172.31.43.7:18280/segment-person", PersonMethod: "person-birefnet-portrait", SubjectMethod: "person-bria-rmbg"}}
	if err := c.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"g6-cloth", "g6-face", "g6-human"} {
		if !ProviderSupports(c.Providers[name], CapabilityText) || ProviderSupports(c.Providers[name], CapabilityImage) {
			t.Fatal("incorrect provider capability")
		}
	}
	y := c.Providers["g6-human"]
	y.HumanYOLO.Username = "user"
	c.Providers["g6-human"] = y
	if err := c.Validate(); err == nil {
		t.Fatal("partial credentials must be rejected")
	}
}
