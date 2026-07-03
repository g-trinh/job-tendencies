package llm

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

// AC: DeepSeek client reuses the same schemas/parsers as Claude, so Extract and
// GenerateAdapter yield parser-equivalent results from an OpenAI-compatible response.
// AC: non-2xx responses are wrapped as errors.
// AC: a response with no tool_call is an error.

// canned OpenAI-compatible tool_call response bodies, keyed by expected tool name.
func toolCallResponse(t *testing.T, toolName string, arguments any) string {
	t.Helper()

	args, err := json.Marshal(arguments)
	if err != nil {
		t.Fatalf("marshalling arguments: %v", err)
	}

	resp := map[string]any{
		"choices": []map[string]any{
			{
				"message": map[string]any{
					"tool_calls": []map[string]any{
						{
							"function": map[string]any{
								"name":      toolName,
								"arguments": json.RawMessage(args),
							},
						},
					},
				},
			},
		},
	}
	body, err := json.Marshal(resp)
	if err != nil {
		t.Fatalf("marshalling response: %v", err)
	}
	return string(body)
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func TestDeepSeekClient_Extract_ParserParity(t *testing.T) {
	t.Parallel()

	fixture := map[string]interface{}{
		"skills":        map[string]interface{}{"value": []string{"Go", "PostgreSQL"}, "confidence": 90},
		"remote_policy": map[string]interface{}{"value": "hybrid", "confidence": 85},
		"office_days":   map[string]interface{}{"value": 2, "confidence": 80},
		"contract_type": map[string]interface{}{"value": "cdi", "confidence": 95},
		"working_days":  map[string]interface{}{"value": "full_time", "confidence": 88},
		"salary_min":    map[string]interface{}{"value": 55000, "confidence": 70},
		"salary_max":    map[string]interface{}{"value": 70000, "confidence": 70},
		"seniority":     map[string]interface{}{"value": "senior", "confidence": 82},
		"recruiter":     map[string]interface{}{"value": nil, "confidence": 0},
		"understanding": 88,
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-key" {
			t.Errorf("Authorization header = %q; want Bearer test-key", r.Header.Get("Authorization"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(toolCallResponse(t, "extract_listing", fixture)))
	}))
	defer server.Close()

	client, err := newDeepSeek("test-key", "deepseek-chat", server.URL, testLogger())
	if err != nil {
		t.Fatalf("newDeepSeek() unexpected error: %v", err)
	}

	listing, err := client.Extract(context.Background(), "raw listing payload")
	if err != nil {
		t.Fatalf("Extract() unexpected error: %v", err)
	}

	if listing.Understanding.Int() != 88 {
		t.Errorf("Understanding = %d; want 88", listing.Understanding.Int())
	}
	if len(listing.Skills.Value) != 2 {
		t.Errorf("Skills.Value len = %d; want 2", len(listing.Skills.Value))
	}
	if listing.SalaryMin.Value == nil || *listing.SalaryMin.Value != 55000 {
		t.Errorf("SalaryMin.Value = %v; want 55000", listing.SalaryMin.Value)
	}
}

func TestDeepSeekClient_GenerateAdapter_Unmarshals(t *testing.T) {
	t.Parallel()

	fixture := map[string]interface{}{
		"board":      "example-board",
		"fetch_mode": "json_api",
		"search": map[string]interface{}{
			"url_template":     "https://example.com/api/search?q={{query}}",
			"method":           "GET",
			"param_map":        map[string]interface{}{"query": "q"},
			"pagination":       map[string]interface{}{},
			"result_node_path": "$.results",
			"result_fields":    map[string]interface{}{"id": "$.id"},
		},
		"listing": map[string]interface{}{
			"fetch":       "use_search_payload",
			"raw_capture": "$",
		},
		"incremental": map[string]interface{}{
			"cursor_field":     "posted_at",
			"overlap_buffer":   "24h",
			"safety_max_pages": 10,
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(toolCallResponse(t, "generate_adapter", fixture)))
	}))
	defer server.Close()

	client, err := newDeepSeek("test-key", "deepseek-chat", server.URL, testLogger())
	if err != nil {
		t.Fatalf("newDeepSeek() unexpected error: %v", err)
	}

	spec, err := client.GenerateAdapter(context.Background(), "https://example.com", "<html></html>")
	if err != nil {
		t.Fatalf("GenerateAdapter() unexpected error: %v", err)
	}
	if spec.Board != "example-board" {
		t.Errorf("Board = %q; want %q", spec.Board, "example-board")
	}
}

func TestDeepSeekClient_DoChatCompletion_NonSuccessStatus(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":"invalid api key"}`))
	}))
	defer server.Close()

	client, err := newDeepSeek("bad-key", "deepseek-chat", server.URL, testLogger())
	if err != nil {
		t.Fatalf("newDeepSeek() unexpected error: %v", err)
	}

	_, err = client.Extract(context.Background(), "raw listing payload")
	if err == nil {
		t.Fatal("Extract() expected error for non-2xx response; got nil")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Extract() error = %q; want it to mention status 401", err.Error())
	}
}

func TestDeepSeekClient_DoChatCompletion_MissingToolCall(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"tool_calls":[]}}]}`))
	}))
	defer server.Close()

	client, err := newDeepSeek("test-key", "deepseek-chat", server.URL, testLogger())
	if err != nil {
		t.Fatalf("newDeepSeek() unexpected error: %v", err)
	}

	_, err = client.Extract(context.Background(), "raw listing payload")
	if err == nil {
		t.Fatal("Extract() expected error when response has no tool_call; got nil")
	}
}

func TestNewDeepSeek_ValidatesRequiredFields(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		apiKey  string
		modelID string
		wantErr bool
	}{
		{name: "missing api key errors", apiKey: "", modelID: "deepseek-chat", wantErr: true},
		{name: "missing model id errors", apiKey: "key", modelID: "", wantErr: true},
		{name: "valid inputs succeed", apiKey: "key", modelID: "deepseek-chat", wantErr: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			_, err := newDeepSeek(tc.apiKey, tc.modelID, "", testLogger())

			if tc.wantErr && err == nil {
				t.Fatal("newDeepSeek() expected error; got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("newDeepSeek() unexpected error: %v", err)
			}
		})
	}
}

func TestPDFToText(t *testing.T) {
	t.Parallel()

	t.Run("returns error for non-pdf bytes", func(t *testing.T) {
		t.Parallel()

		_, err := pdfToText([]byte("not a pdf"))
		if err == nil {
			t.Fatal("pdfToText() expected error for invalid pdf bytes; got nil")
		}
	})

	t.Run("returns error for empty input", func(t *testing.T) {
		t.Parallel()

		_, err := pdfToText([]byte{})
		if err == nil {
			t.Fatal("pdfToText() expected error for empty input; got nil")
		}
	})
}
