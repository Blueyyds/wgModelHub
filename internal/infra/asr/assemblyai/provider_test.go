package assemblyai

import (
	"errors"
	"testing"

	common "github.com/wgdl666/wgModelHub/internal/infra/asr"
)

func TestParseErrorFrameKeepsCode(t *testing.T) {
	_, err := parse(1, []byte(`{"type":"Error","error_code":1008,"error":"account balance is insufficient for a new streaming session"}`))
	var failure *common.SessionFailure
	if !errors.As(err, &failure) || failure.Code != "1008" {
		t.Fatalf("err=%v", err)
	}
	if failure.Reason == "" {
		t.Fatal("reason must stay available for logs")
	}
	code, ok := common.FailureCode(err)
	if !ok || code != "1008" {
		t.Fatalf("metric code=%s ok=%v", code, ok)
	}
}

func TestParseTurnStillReturnsTranscript(t *testing.T) {
	out, err := parse(1, []byte(`{"type":"Turn","transcript":"拍照","end_of_turn":true}`))
	if err != nil || len(out) != 1 || out[0].GetText() != "拍照" {
		t.Fatalf("out=%v err=%v", out, err)
	}
}
