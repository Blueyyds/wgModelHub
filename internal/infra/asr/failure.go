package asr

import (
	"errors"
	"strconv"

	"github.com/gorilla/websocket"
)

// SessionFailure 是供应商在关闭连接前给出的会话拒绝。
// Reason 只进日志，不能当指标标签：账号说明可能很长，也会带账户细节。
type SessionFailure struct {
	Code   string
	Reason string
}

func (e *SessionFailure) Error() string {
	if e == nil {
		return "asr session rejected"
	}
	if e.Reason == "" {
		return "asr session rejected: " + e.Code
	}
	return "asr session rejected: " + e.Code + ": " + e.Reason
}

// FailureCode 取出可打点的数字码。Error 帧优先于随后的关闭帧，避免同一次拒绝计两次。
// 正常关闭和 Going Away 不是供应商拒绝，不能进失败计数。
func FailureCode(err error) (string, bool) {
	var failure *SessionFailure
	if errors.As(err, &failure) {
		return normalizeFailureCode(failure.Code)
	}
	var closeErr *websocket.CloseError
	if errors.As(err, &closeErr) {
		if closeErr.Code == websocket.CloseNormalClosure || closeErr.Code == websocket.CloseGoingAway {
			return "", false
		}
		return normalizeFailureCode(strconv.Itoa(closeErr.Code))
	}
	return "", false
}

func normalizeFailureCode(code string) (string, bool) {
	// 帧里写成 0 或空时没有可用数字码，仍要能按 unknown 聚合，否则账号拒绝会从告警里消失。
	if code == "" || code == "0" || code == "unknown" {
		return "unknown", true
	}
	if len(code) < 3 || len(code) > 4 {
		return "", false
	}
	for _, c := range code {
		if c < '0' || c > '9' {
			return "", false
		}
	}
	return code, true
}
