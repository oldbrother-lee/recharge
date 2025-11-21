package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	pl "recharge-go/internal/service/platform"
	"recharge-go/internal/model"
	"recharge-go/pkg/signature"
)

type submitReq map[string]string

type resp struct{
	Code int `json:"code"`
	Msg string `json:"msg"`
}

func newMockXixiServer(t *testing.T, appID, appSecret, secretKey string, slow bool, status int) *httptest.Server {
	h := http.NewServeMux()
	h.HandleFunc("/api/order/submit", func(w http.ResponseWriter, r *http.Request) {
		if slow { time.Sleep(200 * time.Millisecond) }
		var m map[string]string
		_ = json.NewDecoder(r.Body).Decode(&m)
		// verify sign
		sig := m["sign"]
		if sig == "" || !signature.VerifyXixiSign(m, sig, appSecret, secretKey) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(resp{Code: 401, Msg: "bad sign"})
			return
		}
		if status != http.StatusOK {
			w.WriteHeader(status)
			_ = json.NewEncoder(w).Encode(resp{Code: 500, Msg: "server error"})
			return
		}
		_ = json.NewEncoder(w).Encode(resp{Code: 0, Msg: "ok"})
	})
	h.HandleFunc("/api/order/query", func(w http.ResponseWriter, r *http.Request) {
		var m map[string]string
		_ = json.NewDecoder(r.Body).Decode(&m)
		sig := m["sign"]
		if sig == "" || !signature.VerifyXixiSign(m, sig, appSecret, secretKey) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(resp{Code: 401, Msg: "bad sign"})
			return
		}
		_ = json.NewEncoder(w).Encode(resp{Code: 0, Msg: "ok"})
	})
	return httptest.NewServer(h)
}

func TestXixiClient_Submit_Success(t *testing.T) {
	appID, appSecret, secretKey := "app_001", "secret_001", "sk_001"
	ts := newMockXixiServer(t, appID, appSecret, secretKey, false, http.StatusOK)
	defer ts.Close()

	cli := pl.NewXixiClientWithParams(ts.URL, appID, appSecret, secretKey, 2*time.Second)
	order := &model.Order{OrderNumber: "ORD123", Mobile: "13800138000", ProductID: 1001, Denom: 50}
	params := cli.BuildParams(order)
	b, err := cli.Submit(context.Background(), params)
	if err != nil { t.Fatalf("submit err: %v", err) }
	if !strings.Contains(string(b), "\"code\":0") { t.Fatalf("unexpected resp: %s", string(b)) }
}

func TestXixiClient_Submit_HTTPError(t *testing.T) {
	appID, appSecret, secretKey := "app_001", "secret_001", "sk_001"
	ts := newMockXixiServer(t, appID, appSecret, secretKey, false, http.StatusInternalServerError)
	defer ts.Close()
	cli := pl.NewXixiClientWithParams(ts.URL, appID, appSecret, secretKey, 2*time.Second)
	order := &model.Order{OrderNumber: "ORD123", Mobile: "13800138000", ProductID: 1001, Denom: 50}
	params := cli.BuildParams(order)
	_, err := cli.Submit(context.Background(), params)
	if err == nil || !strings.Contains(err.Error(), "http 500") {
		t.Fatalf("expect http 500 error, got %v", err)
	}
}

func TestXixiClient_Submit_Timeout(t *testing.T) {
	appID, appSecret, secretKey := "app_001", "secret_001", "sk_001"
	ts := newMockXixiServer(t, appID, appSecret, secretKey, true, http.StatusOK)
	defer ts.Close()
	cli := pl.NewXixiClientWithParams(ts.URL, appID, appSecret, secretKey, 50*time.Millisecond)
	order := &model.Order{OrderNumber: "ORD123", Mobile: "13800138000", ProductID: 1001, Denom: 50}
	params := cli.BuildParams(order)
	_, err := cli.Submit(context.Background(), params)
	if err == nil { t.Fatalf("expect timeout error") }
	if !strings.Contains(err.Error(), "Client.Timeout") && !strings.Contains(err.Error(), "context deadline") {
		t.Fatalf("unexpected timeout error: %v", err)
	}
}

func TestXixiClient_Query_Success(t *testing.T) {
	appID, appSecret, secretKey := "app_001", "secret_001", "sk_001"
	ts := newMockXixiServer(t, appID, appSecret, secretKey, false, http.StatusOK)
	defer ts.Close()
	cli := pl.NewXixiClientWithParams(ts.URL, appID, appSecret, secretKey, 2*time.Second)
	b, err := cli.Query(context.Background(), "ORD123")
	if err != nil { t.Fatalf("query err: %v", err) }
	if !strings.Contains(string(b), "\"code\":0") { t.Fatalf("unexpected resp: %s", string(b)) }
}