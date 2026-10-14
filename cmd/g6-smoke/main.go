// g6-smoke exercises the real adapters through an in-memory ModelHub gRPC server.
// It never changes Nacos or a running deployment, and does not need database/telemetry credentials.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/wgdl666/wgModelHub/config"
	modelhubv2 "github.com/wgdl666/wgModelHub/gen/wg_model_hub/v2"
	"github.com/wgdl666/wgModelHub/internal/infra/factory"
	service "github.com/wgdl666/wgModelHub/internal/service/modelhub"
	"github.com/wgdl666/wgModelHub/models"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"gopkg.in/yaml.v3"
)

func main() {
	configFile := flag.String("config", "config/aws-g6.providers.yaml", "G6 provider overlay")
	imageFile := flag.String("image", "", "real portrait image")
	outDir := flag.String("out", "artifacts/g6-smoke", "result directory")
	flag.Parse()
	if err := run(*configFile, *imageFile, *outDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func run(configFile, imageFile, outDir string) error {
	raw, err := os.ReadFile(imageFile)
	if err != nil {
		return err
	}
	content, err := os.ReadFile(configFile)
	if err != nil {
		return err
	}
	var cfg config.Config
	if err := yaml.Unmarshal(content, &cfg); err != nil {
		return err
	}
	// Validation requires service configuration, but the isolated RPC server opens neither DB nor telemetry.
	cfg.Database.DSN = "postgres://unused"
	cfg.Logfire.Token = "not-used"
	cfg.Logfire.Env = "dev"
	cfg.Logfire.Service = "g6-smoke"
	cfg.Server.ListenAddress = ":50053"
	cfg.Server.HTTPListenAddress = ":51053"
	if err := cfg.Validate(); err != nil {
		return err
	}
	sets, err := factory.Build(context.Background(), cfg)
	if err != nil {
		return err
	}
	server := grpc.NewServer(grpc.MaxRecvMsgSize(64<<20), grpc.MaxSendMsgSize(64<<20))
	modelhubv2.RegisterModelHubServiceServer(server, service.New(config.NewLiveConfig(cfg), sets, nil))
	listener := bufconn.Listen(64 << 20)
	defer listener.Close()
	defer server.Stop()
	go server.Serve(listener)
	conn, err := grpc.NewClient("passthrough:///g6-smoke", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) { return listener.DialContext(ctx) }), grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(64<<20), grpc.MaxCallSendMsgSize(64<<20)))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := modelhubv2.NewModelHubServiceClient(conn)
	if err := os.MkdirAll(outDir, 0750); err != nil {
		return err
	}
	results := []map[string]any{}
	for _, model := range []string{models.ClothYOLO, models.HumanYOLO, models.HumanParser, models.HumanParse, models.FaceSim, models.SegmentPersonBria, models.SegmentSubjectBria} {
		count := 1
		if model == models.FaceSim {
			count = 2
		}
		parts := []*modelhubv2.ContentPart{}
		for i := 0; i < count; i++ {
			parts = append(parts, &modelhubv2.ContentPart{Content: &modelhubv2.ContentPart_Image{Image: &modelhubv2.Media{MimeType: "image/jpeg", Source: &modelhubv2.Media_Data{Data: raw}}}})
		}
		request := &modelhubv2.GenerateRequest{Model: model, Input: &modelhubv2.Input{Items: []*modelhubv2.InputItem{{Item: &modelhubv2.InputItem_Message{Message: &modelhubv2.Message{Role: modelhubv2.Role_ROLE_USER, Parts: parts}}}}}}
		imageOutput := model == models.SegmentPersonBria || model == models.SegmentSubjectBria
		if imageOutput {
			request.Output = &modelhubv2.OutputSpec{Kind: &modelhubv2.OutputSpec_Image{Image: &modelhubv2.ImageOutput{}}}
		} else {
			request.Output = &modelhubv2.OutputSpec{Kind: &modelhubv2.OutputSpec_Text{Text: &modelhubv2.TextOutput{}}}
		}
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
		start := time.Now()
		stream, err := client.Generate(ctx, request)
		if err != nil {
			cancel()
			return fmt.Errorf("%s: %w", model, err)
		}
		finalCount := 0
		items := []*modelhubv2.OutputItem{}
		for {
			event, err := stream.Recv()
			if err == io.EOF {
				break
			}
			if err != nil {
				cancel()
				return fmt.Errorf("%s: %w", model, err)
			}
			if event.Final {
				finalCount++
			}
			items = append(items, event.Items...)
		}
		cancel()
		if finalCount != 1 || len(items) == 0 {
			return fmt.Errorf("%s: invalid final result", model)
		}
		result := map[string]any{"model": model, "elapsed_ms": float64(time.Since(start).Milliseconds()), "success": true}
		if imageOutput {
			data := items[0].GetImage().GetData()
			im, err := png.Decode(bytes.NewReader(data))
			if err != nil {
				return err
			}
			transparent, opaque := false, false
			for y := im.Bounds().Min.Y; y < im.Bounds().Max.Y; y++ {
				for x := im.Bounds().Min.X; x < im.Bounds().Max.X; x++ {
					_, _, _, a := im.At(x, y).RGBA()
					transparent = transparent || a == 0
					opaque = opaque || a == 65535
				}
			}
			if !transparent || !opaque {
				return fmt.Errorf("%s: no valid transparent cutout", model)
			}
			result["dimensions"] = []int{im.Bounds().Dx(), im.Bounds().Dy()}
			result["png_bytes"] = len(data)
			if err := os.WriteFile(filepath.Join(outDir, model+".png"), data, 0640); err != nil {
				return err
			}
		} else {
			data := []byte(items[0].GetText())
			var j map[string]any
			if err := json.Unmarshal(data, &j); err != nil {
				return err
			}
			switch model {
			case models.ClothYOLO:
				if j["success"] != true || len(j["items"].([]any)) == 0 {
					return fmt.Errorf("no clothes found")
				}
				result["items"] = len(j["items"].([]any))
			case models.HumanYOLO:
				if j["success"] != true || j["person_count"].(float64) < 1 {
					return fmt.Errorf("no person found")
				}
				result["person_count"] = j["person_count"]
			case models.HumanParser, models.HumanParse:
				if j["success"] != true || len(j["id2label"].(map[string]any)) != 18 {
					return fmt.Errorf("invalid parsing result")
				}
				result["pred_shape"] = j["pred_shape"]
				result["labels"] = 18
			case models.FaceSim:
				if j["similarity"].(float64) < .99 {
					return fmt.Errorf("self comparison failed")
				}
				result["self_similarity"] = j["similarity"]
			}
			if err := os.WriteFile(filepath.Join(outDir, model+".json"), data, 0640); err != nil {
				return err
			}
		}
		results = append(results, result)
		b, _ := json.Marshal(result)
		fmt.Println(string(b))
	}
	report, _ := json.MarshalIndent(map[string]any{"transport": "ModelHub gRPC loopback -> real AWS-G6 services", "results": results}, "", "  ")
	return os.WriteFile(filepath.Join(outDir, "verification.json"), report, 0640)
}
