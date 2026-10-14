package rekognitionfaces

import (
	"bytes"
	"context"
	"encoding/json"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	modelhubv2 "github.com/wgdl666/wgModelHub/gen/wg_model_hub/v2"
)

func TestDetectFacesOmitsGenderUnlessRequested(t *testing.T) {
	stub := &stubFaceAPI{out: faceBoxOnly()}
	provider := &Provider{name: "rekognition", detect: true, client: stub}
	event, err := provider.detectFaces(context.Background(), detectRequest(t, ""))
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.in.Attributes) != 0 {
		t.Fatalf("default detect attributes=%v", stub.in.Attributes)
	}
	var decoded struct {
		Faces []faceBox `json:"faces"`
	}
	if err := json.Unmarshal([]byte(event.GetItems()[0].GetText()), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Faces) != 1 || decoded.Faces[0].Gender != "" {
		t.Fatalf("default faces=%+v", decoded.Faces)
	}
}

func TestDetectFacesReturnsGenderWhenAttributesAll(t *testing.T) {
	stub := &stubFaceAPI{out: faceWithGender(types.GenderTypeFemale, 96)}
	provider := &Provider{name: "rekognition", detect: true, client: stub}
	event, err := provider.detectFaces(context.Background(), detectRequest(t, `{"max_faces":10,"attributes":"all"}`))
	if err != nil {
		t.Fatal(err)
	}
	if len(stub.in.Attributes) != 1 || stub.in.Attributes[0] != types.AttributeAll {
		t.Fatalf("attributes=%v", stub.in.Attributes)
	}
	var decoded struct {
		Faces []faceBox `json:"faces"`
	}
	if err := json.Unmarshal([]byte(event.GetItems()[0].GetText()), &decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded.Faces) != 1 || decoded.Faces[0].Gender != "female" || decoded.Faces[0].GenderConfidence != 96 {
		t.Fatalf("faces=%+v", decoded.Faces)
	}
}

func faceBoxOnly() *rekognition.DetectFacesOutput {
	left, top, width, height := float32(0), float32(0), float32(1), float32(1)
	return &rekognition.DetectFacesOutput{FaceDetails: []types.FaceDetail{{
		BoundingBox: &types.BoundingBox{Left: &left, Top: &top, Width: &width, Height: &height},
	}}}
}

func faceWithGender(value types.GenderType, confidence float32) *rekognition.DetectFacesOutput {
	left, top, width, height := float32(0), float32(0), float32(1), float32(1)
	return &rekognition.DetectFacesOutput{FaceDetails: []types.FaceDetail{{
		BoundingBox: &types.BoundingBox{Left: &left, Top: &top, Width: &width, Height: &height},
		Gender:      &types.Gender{Value: value, Confidence: &confidence},
	}}}
}

func detectRequest(t *testing.T, text string) *modelhubv2.GenerateRequest {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.RGBA{A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	parts := []*modelhubv2.ContentPart{{
		Content: &modelhubv2.ContentPart_Image{Image: &modelhubv2.Media{
			MimeType: "image/png",
			Source:   &modelhubv2.Media_Data{Data: buf.Bytes()},
		}},
	}}
	if text != "" {
		parts = append(parts, &modelhubv2.ContentPart{Content: &modelhubv2.ContentPart_Text{Text: text}})
	}
	return &modelhubv2.GenerateRequest{Input: &modelhubv2.Input{Items: []*modelhubv2.InputItem{{
		Item: &modelhubv2.InputItem_Message{Message: &modelhubv2.Message{
			Role: modelhubv2.Role_ROLE_USER, Parts: parts,
		}},
	}}}}
}

type stubFaceAPI struct {
	in  *rekognition.DetectFacesInput
	out *rekognition.DetectFacesOutput
}

func (s *stubFaceAPI) DetectFaces(_ context.Context, params *rekognition.DetectFacesInput, _ ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
	s.in = params
	return s.out, nil
}

func (s *stubFaceAPI) SearchFacesByImage(context.Context, *rekognition.SearchFacesByImageInput, ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
	return nil, nil
}

func (s *stubFaceAPI) IndexFaces(context.Context, *rekognition.IndexFacesInput, ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
	return nil, nil
}

func (s *stubFaceAPI) ListFaces(context.Context, *rekognition.ListFacesInput, ...func(*rekognition.Options)) (*rekognition.ListFacesOutput, error) {
	return nil, nil
}

func (s *stubFaceAPI) DeleteFaces(context.Context, *rekognition.DeleteFacesInput, ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error) {
	return nil, nil
}
