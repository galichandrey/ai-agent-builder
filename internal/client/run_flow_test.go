package client

import (
	"encoding/json"
	"testing"
)

func TestExtractLastMessage(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		want string
	}{
		{
			name: "messages key wins last",
			raw:  `{"outputs":[{"results":{"response":{"text":"A"}}},{"messages":[{"message":"B"}]}]}`,
			want: "B",
		},
		{
			name: "nested text only",
			raw:  `{"outputs":[{"results":{"r":{"text":"hello"}}}]}`,
			want: "hello",
		},
		{
			name: "empty strings skipped",
			raw:  `{"outputs":[{"messages":[{"message":""},{"message":"real"}]}]}`,
			want: "real",
		},
		{
			name: "no message keys -> empty",
			raw:  `{"foo":"bar"}`,
			want: "",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := ExtractLastMessage(json.RawMessage(tc.raw))
			if got != tc.want {
				t.Fatalf("want %q got %q", tc.want, got)
			}
		})
	}
}

func TestRunFlow_PayloadShape(t *testing.T) {
	// проверяем, что клиент шлёт POST на /run/{id} без трейлинг-слэша
	p := RunFlowPayload{InputValue: "x", SessionID: "s"}
	b, _ := json.Marshal(p)
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	if m["input_value"] != "x" || m["session_id"] != "s" {
		t.Fatalf("unexpected payload: %s", b)
	}
	if _, ok := m["tweaks"]; ok {
		t.Fatalf("nil tweaks must be omitted")
	}
}
