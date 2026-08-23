package client

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateFlowFromRawPostsDataVerbatim(t *testing.T) {
	dataRaw := json.RawMessage(`{"nodes":[{"id":"n1","extra_native_field":true}],"edges":[],"viewport":{"x":1,"y":2,"zoom":3}}`)
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/flows/" {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		b, _ := io.ReadAll(r.Body)
		json.Unmarshal(b, &gotBody)
		w.WriteHeader(201)
		json.NewEncoder(w).Encode(map[string]any{"id": "flow-1", "name": gotBody["name"]})
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	flow, err := c.CreateFlowFromRaw(context.Background(), "My Flow", "desc", dataRaw)
	if err != nil {
		t.Fatalf("CreateFlowFromRaw: %v", err)
	}
	if flow.ID != "flow-1" {
		t.Errorf("ID = %q", flow.ID)
	}
	dataMap, ok := gotBody["data"].(map[string]any)
	if !ok {
		t.Fatalf("data not object: %T", gotBody["data"])
	}
	nodes, _ := dataMap["nodes"].([]any)
	n0, _ := nodes[0].(map[string]any)
	if n0["extra_native_field"] != true {
		t.Errorf("native field lost: %v", n0)
	}
	vp, _ := dataMap["viewport"].(map[string]any)
	if vp["zoom"] != float64(3) {
		t.Errorf("viewport lost: %v", vp)
	}
}

func TestGetFlowRawExtractsDataVerbatim(t *testing.T) {
	full := `{"id":"f9","name":"X","data":{"nodes":[{"id":"a","weird":{"deep":[1,2,{"k":"v"}]}}],"edges":[]}}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || !strings.HasSuffix(r.URL.Path, "/flows/f9") {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		w.WriteHeader(200)
		io.WriteString(w, full)
	}))
	defer srv.Close()

	c := NewClient(testConfig(srv.URL))
	raw, err := c.GetFlowRaw(context.Background(), "f9")
	if err != nil {
		t.Fatalf("GetFlowRaw: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(raw, &data); err != nil {
		t.Fatalf(".data not returned: %s", string(raw))
	}
	nodes, _ := data["nodes"].([]any)
	n0, _ := nodes[0].(map[string]any)
	if _, ok := n0["weird"]; !ok {
		t.Error("unknown nested field lost")
	}
}

func TestFlowRawRoundTrip(t *testing.T) {
	dataRaw := json.RawMessage(`{"nodes":[{"x":1}],"edges":[],"viewport":null}`)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{"id": "rt", "name": "rt"})
	}))
	defer srv.Close()
	c := NewClient(testConfig(srv.URL))
	if _, err := c.CreateFlowFromRaw(context.Background(), "n", "d", dataRaw); err != nil {
		t.Fatal(err)
	}
}
