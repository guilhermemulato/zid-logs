package licensing

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"golang.org/x/crypto/hkdf"
)

const (
	SocketPath     = "/var/run/zid-packages.sock"
	masterSecret   = "zid-packages-master-secret-2026-01"
	hkdfInfo       = "zid-packages-hkdf"
	opCheck        = "CHECK"
	defaultTimeout = 3 * time.Second
)

var (
	ErrInvalidLicense = errors.New("licenca invalida")
	ErrBadSignature   = errors.New("bad_sig")
	ErrIPCUnavailable = errors.New("ipc indisponivel")
)

type Request struct {
	Op      string `json:"op"`
	Package string `json:"package"`
	TS      int64  `json:"ts"`
	Nonce   string `json:"nonce"`
	Sig     string `json:"sig"`
}

type Response struct {
	OK         bool   `json:"ok"`
	Licensed   bool   `json:"licensed"`
	Mode       string `json:"mode"`
	ValidUntil int64  `json:"valid_until"`
	Reason     string `json:"reason"`
	TS         int64  `json:"ts"`
	Sig        string `json:"sig"`
}

func Check(pkg string) (Response, error) {
	pkg = strings.TrimSpace(pkg)
	if pkg == "" {
		return Response{}, errors.New("package vazio")
	}

	key, err := deriveKey()
	if err != nil {
		return Response{}, err
	}

	nonce, err := randomNonce()
	if err != nil {
		return Response{}, err
	}

	req, err := buildRequestWithKey(pkg, time.Now().Unix(), nonce, key)
	if err != nil {
		return Response{}, err
	}

	resp, err := sendRequest(req)
	if err != nil {
		return Response{}, err
	}

	if err := verifyResponse(key, resp); err != nil {
		return resp, err
	}

	if !resp.OK {
		reason := strings.TrimSpace(resp.Reason)
		if reason == "" {
			reason = "ok=false"
		}
		return resp, fmt.Errorf("%w: %s", ErrInvalidLicense, reason)
	}

	if !resp.Licensed {
		reason := strings.TrimSpace(resp.Reason)
		if reason == "" {
			reason = "not licensed"
		}
		return resp, fmt.Errorf("%w: %s", ErrInvalidLicense, reason)
	}

	return resp, nil
}

func sendRequest(req Request) (Response, error) {
	conn, err := net.DialTimeout("unix", SocketPath, defaultTimeout)
	if err != nil {
		return Response{}, fmt.Errorf("%w: %v", ErrIPCUnavailable, err)
	}
	defer conn.Close()

	_ = conn.SetDeadline(time.Now().Add(defaultTimeout))

	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return Response{}, err
	}

	var resp Response
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return Response{}, err
	}

	return resp, nil
}

func verifyResponse(key []byte, resp Response) error {
	if strings.TrimSpace(resp.Sig) == "" {
		return errors.New("resposta sem assinatura")
	}
	payload := resp
	payload.Sig = ""
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	sigBytes, err := hex.DecodeString(resp.Sig)
	if err != nil {
		return fmt.Errorf("assinatura invalida: %w", err)
	}

	expected := sign(key, raw)
	if !hmac.Equal(expected, sigBytes) {
		return ErrBadSignature
	}

	return nil
}

func buildRequestWithKey(pkg string, ts int64, nonce string, key []byte) (Request, error) {
	req := Request{
		Op:      opCheck,
		Package: pkg,
		TS:      ts,
		Nonce:   nonce,
		Sig:     "",
	}
	payload, err := json.Marshal(req)
	if err != nil {
		return Request{}, err
	}
	req.Sig = signHex(key, payload)
	return req, nil
}

func signHex(key, payload []byte) string {
	return hex.EncodeToString(sign(key, payload))
}

func sign(key, payload []byte) []byte {
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	return mac.Sum(nil)
}

func randomNonce() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func deriveKey() ([]byte, error) {
	host, err := shortHostname()
	if err != nil {
		return nil, err
	}
	uid, err := uniqueID()
	if err != nil {
		return nil, err
	}
	salt := []byte(uid + ":" + host)
	reader := hkdf.New(sha256.New, []byte(masterSecret), salt, []byte(hkdfInfo))
	key := make([]byte, 32)
	if _, err := io.ReadFull(reader, key); err != nil {
		return nil, err
	}
	return key, nil
}

func shortHostname() (string, error) {
	host, err := os.Hostname()
	if err != nil {
		return "", err
	}
	host = strings.TrimSpace(host)
	if host == "" {
		return "", errors.New("hostname vazio")
	}
	if idx := strings.IndexByte(host, '.'); idx > 0 {
		host = host[:idx]
	}
	return host, nil
}

func uniqueID() (string, error) {
	raw, err := os.ReadFile("/var/db/uniqueid")
	if err != nil {
		return "", err
	}
	uid := strings.TrimSpace(string(raw))
	if uid == "" {
		return "", errors.New("uniqueid vazio")
	}
	return uid, nil
}
