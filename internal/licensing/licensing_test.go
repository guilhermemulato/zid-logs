package licensing

import (
	"encoding/json"
	"testing"
)

func TestBuildRequestWithKeySignsPayload(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	pkg := "zid-logs"
	ts := int64(1700000000)
	nonce := "abc123"

	req, err := buildRequestWithKey(pkg, ts, nonce, key)
	if err != nil {
		t.Fatalf("buildRequestWithKey error: %v", err)
	}

	if req.Op != opCheck {
		t.Fatalf("unexpected op: %s", req.Op)
	}
	if req.Sig == "" {
		t.Fatalf("sig vazio")
	}

	payload := req
	payload.Sig = ""
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json marshal error: %v", err)
	}

	expected := signHex(key, raw)
	if req.Sig != expected {
		t.Fatalf("sig mismatch: got %s want %s", req.Sig, expected)
	}
}

func TestVerifyResponse(t *testing.T) {
	key := []byte("01234567890123456789012345678901")
	resp := Response{
		OK:         true,
		Licensed:   true,
		Mode:       "OK",
		ValidUntil: 1700600000,
		Reason:     "valid",
		TS:         1700000001,
		Sig:        "",
	}

	payload := resp
	payload.Sig = ""
	raw, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json marshal error: %v", err)
	}
	resp.Sig = signHex(key, raw)

	if err := verifyResponse(key, resp); err != nil {
		t.Fatalf("verifyResponse error: %v", err)
	}

	resp.Sig = "00"
	if err := verifyResponse(key, resp); err == nil {
		t.Fatalf("expected error for bad sig")
	}
}
