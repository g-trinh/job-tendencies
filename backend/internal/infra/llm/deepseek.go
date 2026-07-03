package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ledongthuc/pdf"

	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// DefaultDeepSeekBaseURL is the default DeepSeek API base URL used when no
// base URL is configured.
const DefaultDeepSeekBaseURL = "https://api.deepseek.com"

// deepSeekClient implements domain/llm.AdapterGenerator, domain/llm.ListingExtractor,
// and the profiles.IdentityExtractor port using the DeepSeek OpenAI-compatible
// chat completions API. DeepSeek is text-only, so ExtractIdentity converts the
// LinkedIn PDF to text before calling the API (ADR-006).
type deepSeekClient struct {
	http    *http.Client
	apiKey  string
	baseURL string
	modelID string
	logger  *slog.Logger
}

// newDeepSeek constructs a DeepSeek LLM client. apiKey and modelID are required.
// When baseURL is empty, DefaultDeepSeekBaseURL is used.
func newDeepSeek(apiKey string, modelID string, baseURL string, logger *slog.Logger) (*deepSeekClient, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("deepseek: api key is required")
	}
	if modelID == "" {
		return nil, fmt.Errorf("deepseek: model id is required")
	}
	if baseURL == "" {
		baseURL = DefaultDeepSeekBaseURL
	}
	return &deepSeekClient{
		http:    &http.Client{},
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		modelID: modelID,
		logger:  logger,
	}, nil
}

// openAIMessage is a single chat message in the OpenAI-compatible request format.
type openAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// openAIFunction describes a callable tool's name, description, and JSON Schema
// parameters, in the OpenAI-compatible tool format.
type openAIFunction struct {
	Name        string         `json:"name"`
	Description string         `json:"description,omitempty"`
	Parameters  map[string]any `json:"parameters"`
}

// openAITool wraps an openAIFunction as a "function" typed tool.
type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

// chatCompletionRequest is the OpenAI-compatible chat/completions request body.
// ToolChoice is "auto" (not a forced named function): DeepSeek's thinking-mode
// models reject a forced tool_choice ("Thinking mode does not support this
// tool_choice"). With a single declared tool and an instructing system prompt the
// model reliably calls it; if it ever does not, doChatCompletion errors cleanly.
type chatCompletionRequest struct {
	Model      string          `json:"model"`
	Messages   []openAIMessage `json:"messages"`
	Tools      []openAITool    `json:"tools"`
	ToolChoice string          `json:"tool_choice"`
	MaxTokens  int             `json:"max_tokens,omitempty"`
}

// chatCompletionResponse is the OpenAI-compatible chat/completions response body.
type chatCompletionResponse struct {
	Choices []struct {
		Message struct {
			ToolCalls []struct {
				Function struct {
					Name      string          `json:"name"`
					Arguments json.RawMessage `json:"arguments"`
				} `json:"function"`
			} `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

// doChatCompletion POSTs a single-tool chat completion request (tool_choice "auto")
// to the DeepSeek API and returns the arguments of the first tool call in the
// response. toolName names the one declared tool; schema is its JSON Schema
// "properties"/"required" pair, wrapped as a type:"object" parameters document.
func (c *deepSeekClient) doChatCompletion(ctx context.Context, systemPrompt, userMsg, toolName string, properties map[string]interface{}, required []string) (json.RawMessage, error) {
	reqBody := chatCompletionRequest{
		Model: c.modelID,
		Messages: []openAIMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userMsg},
		},
		Tools: []openAITool{
			{
				Type: "function",
				Function: openAIFunction{
					Name: toolName,
					Parameters: map[string]any{
						"type":       "object",
						"properties": properties,
						"required":   required,
					},
				},
			},
		},
		ToolChoice: "auto",
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("marshalling deepseek request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("building deepseek request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling deepseek chat completions: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepseek chat completions returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var completion chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completion); err != nil {
		return nil, fmt.Errorf("decoding deepseek response: %w", err)
	}

	if len(completion.Choices) == 0 || len(completion.Choices[0].Message.ToolCalls) == 0 {
		return nil, fmt.Errorf("deepseek response contains no tool_call")
	}

	// OpenAI-compatible APIs (DeepSeek included) encode function.arguments as a
	// JSON *string* (e.g. "{\"skills\":[...]}"), not a nested object. Unwrap it to
	// the inner JSON so the shared parsers — which expect a raw object, as Claude's
	// tool input is — can decode it. Fall back to the raw bytes if a server returns
	// the object directly.
	raw := completion.Choices[0].Message.ToolCalls[0].Function.Arguments
	var argStr string
	if err := json.Unmarshal(raw, &argStr); err == nil {
		return json.RawMessage(argStr), nil
	}
	return raw, nil
}

// GenerateAdapter calls DeepSeek to produce a declarative AdapterSpec from a board URL
// and an example page response. The result is data only — it must be human-reviewed
// and approved before the scraper evaluates it.
func (c *deepSeekClient) GenerateAdapter(ctx context.Context, boardURL string, exampleResponse string) (*domainllm.AdapterSpec, error) {
	spec := adapterSpecSchema()
	userMsg := fmt.Sprintf("Board URL: %s\n\nExample response:\n%s", boardURL, exampleResponse)

	raw, err := c.doChatCompletion(ctx, adapterSystemPrompt, userMsg, "generate_adapter", spec.properties, spec.required)
	if err != nil {
		return nil, fmt.Errorf("calling deepseek for adapter generation: %w", err)
	}

	var adapterSpec domainllm.AdapterSpec
	if err := json.Unmarshal(raw, &adapterSpec); err != nil {
		return nil, fmt.Errorf("unmarshalling adapter spec: %w", err)
	}

	c.logger.InfoContext(ctx, "adapter generated", "board_url", boardURL, "provider", "deepseek")
	return &adapterSpec, nil
}

// Extract calls DeepSeek to extract structured fields from a raw job listing payload.
// Each returned field carries a per-field confidence score and the listing carries
// an overall understanding score.
func (c *deepSeekClient) Extract(ctx context.Context, raw string) (*domainllm.ExtractedListing, error) {
	toolInput, err := c.doChatCompletion(ctx, extractionSystemPrompt, raw, "extract_listing", extractionProperties, extractionRequired)
	if err != nil {
		return nil, fmt.Errorf("calling deepseek for extraction: %w", err)
	}

	listing, err := parseExtractedListing(toolInput)
	if err != nil {
		return nil, fmt.Errorf("parsing extracted listing: %w", err)
	}

	c.logger.InfoContext(ctx, "listing extracted",
		"understanding", listing.Understanding.Int(),
		"model", c.modelID,
		"provider", "deepseek",
	)
	return listing, nil
}

// ExtractIdentity converts the LinkedIn PDF export to text and sends it to DeepSeek
// for professional identity extraction (skills, raw experience, seniority). DeepSeek
// is text-only, so the PDF is converted before the request — unlike the Claude client,
// which sends the PDF natively as a document content block.
//
// Use this to populate a profile's identity on first import. The caller is
// responsible for enforcing the single-import guard before calling.
func (c *deepSeekClient) ExtractIdentity(ctx context.Context, pdfBytes []byte) (*domainllm.ExtractedIdentity, error) {
	text, err := pdfToText(pdfBytes)
	if err != nil {
		return nil, fmt.Errorf("converting pdf to text: %w", err)
	}

	userMsg := "Extract professional identity from this LinkedIn PDF export (converted to text below):\n\n" + text

	raw, err := c.doChatCompletion(ctx, identitySystemPrompt, userMsg, "extract_identity", identityProperties, identityRequired)
	if err != nil {
		return nil, fmt.Errorf("calling deepseek for identity extraction: %w", err)
	}

	identity, err := parseExtractedIdentity(raw)
	if err != nil {
		return nil, fmt.Errorf("parsing extracted identity: %w", err)
	}

	c.logger.InfoContext(ctx, "identity extracted", "model", c.modelID, "provider", "deepseek")
	return identity, nil
}

// pdfToText extracts the plain text content of a PDF document. It recovers from
// panics raised by the underlying pdf library (a known failure mode on malformed
// PDFs) and returns them as errors. Returns an error when the extracted text is
// empty, since an empty identity extraction request is never useful.
func pdfToText(pdfBytes []byte) (text string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("pdf library panicked: %v", r)
		}
	}()

	reader, err := pdf.NewReader(bytes.NewReader(pdfBytes), int64(len(pdfBytes)))
	if err != nil {
		return "", fmt.Errorf("reading pdf: %w", err)
	}

	var sb strings.Builder
	for i := 1; i <= reader.NumPage(); i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}
		pageText, textErr := page.GetPlainText(nil)
		if textErr != nil {
			return "", fmt.Errorf("extracting text from page %d: %w", i, textErr)
		}
		sb.WriteString(pageText)
	}

	text = sb.String()
	if strings.TrimSpace(text) == "" {
		return "", fmt.Errorf("pdf contains no extractable text")
	}
	return text, nil
}
