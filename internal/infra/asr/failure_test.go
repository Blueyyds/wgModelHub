package asr

import (
	"testing"

	"github.com/gorilla/websocket"
)

func TestFailureCodePrefersSessionErrorOverClose(t *testing.T) {
	code, ok := FailureCode(&SessionFailure{Code: "1008", Reason: "account balance is insufficient for a new streaming session"})
	if !ok || code != "1008" {
		t.Fatalf("code=%s ok=%v", code, ok)
	}
	code, ok = FailureCode(&websocket.CloseError{Code: websocket.ClosePolicyViolation, Text: "See Error message for details"})
	if !ok || code != "1008" {
		t.Fatalf("close code=%s ok=%v", code, ok)
	}
	if _, ok := FailureCode(&websocket.CloseError{Code: websocket.CloseNormalClosure}); ok {
		t.Fatal("normal close must not count as a session failure")
	}
	if _, ok := FailureCode(&SessionFailure{Code: "see details"}); ok {
		t.Fatal("non-numeric reason must not become a label")
	}
}
