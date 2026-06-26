// Package llm implements the domain llm port using the Anthropic Go SDK.
// Both AdapterGenerator and ListingExtractor are implemented by a single Client
// to allow prompt-caching configuration to be shared. The stable system prompt
// and extraction JSON schema carry cache-control breakpoints, reducing token cost
// on repeated extraction calls.
//
// Wiring: construct one Client per binary in cmd/; pass it to both interfaces.
package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"

	"github.com/g-trinh/job-tendencies/internal/domain/kernel"
	domainllm "github.com/g-trinh/job-tendencies/internal/domain/llm"
)

// DefaultModelID is the default Claude model used when no model id is configured.
const DefaultModelID = "claude-opus-4-8"

// extractionSystemPrompt is the stable system prompt used for listing extraction.
// It is sent with a cache-control breakpoint so it is cached across repeated calls.
const extractionSystemPrompt = `You are an expert at extracting structured information from job listings.
Given a raw job listing (in HTML or JSON format, in French or English), extract the requested fields.
For each field, provide:
- "value": the extracted value (use null when the field is absent or cannot be determined)
- "confidence": an integer 0-100 representing how certain you are about this value
  (0 = absent/unrecognisable, 100 = clearly stated)

Also provide a top-level "understanding" score (0-100) for how well you could parse the
listing overall (0 = incomprehensible, 100 = fully structured and clear).

Rules:
- Never translate raw text; return values in their original language.
- Salary values should be annual totals in whole euros (no cents).
- Remote policy values: "on_site", "hybrid", or "full_remote" only.
- Contract type values: "cdi", "cdd", "freelance", or "interim" only.
- Working days values: "full_time", "part_time", or "four_day" only.
- Seniority values: "entry", "mid", "senior", "lead", or "exec" only.
- If a field is absent, set value to null and confidence to 0.`

// adapterSystemPrompt is the stable system prompt used for adapter generation.
const adapterSystemPrompt = `You are an expert at analysing job board websites and generating scraping configurations.
Given a board URL and an example API response or HTML page, generate a declarative scraping adapter spec.
The spec must be pure data (JSONPath expressions, CSS selectors, URL templates) — never code.
Output must be valid JSON matching the AdapterSpec schema exactly.`

// extractionSchema is the JSON schema passed to Claude as a tool, defining the shape
// of the ExtractedListing. Sent with a cache-control breakpoint alongside the system prompt.
var extractionSchema = map[string]interface{}{
	"type": "object",
	"properties": map[string]interface{}{
		"skills": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"type": "array", "items": map[string]interface{}{"type": "string"}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"remote_policy": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"type": "string", "enum": []string{"on_site", "hybrid", "full_remote"}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"office_days": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 7},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"contract_type": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"type": "string", "enum": []string{"cdi", "cdd", "freelance", "interim"}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"working_days": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"type": "string", "enum": []string{"full_time", "part_time", "four_day"}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"salary_min": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"oneOf": []interface{}{map[string]interface{}{"type": "integer"}, map[string]interface{}{"type": "null"}}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"salary_max": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"oneOf": []interface{}{map[string]interface{}{"type": "integer"}, map[string]interface{}{"type": "null"}}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"seniority": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value":      map[string]interface{}{"type": "string", "enum": []string{"entry", "mid", "senior", "lead", "exec"}},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"recruiter": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"value": map[string]interface{}{
					"oneOf": []interface{}{
						map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"name":         map[string]interface{}{"type": "string"},
								"email":        map[string]interface{}{"type": "string"},
								"linkedin_url": map[string]interface{}{"type": "string"},
								"phone":        map[string]interface{}{"type": "string"},
							},
						},
						map[string]interface{}{"type": "null"},
					},
				},
				"confidence": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
			},
			"required": []string{"value", "confidence"},
		},
		"understanding": map[string]interface{}{"type": "integer", "minimum": 0, "maximum": 100},
	},
	"required": []string{
		"skills", "remote_policy", "office_days", "contract_type", "working_days",
		"salary_min", "salary_max", "seniority", "recruiter", "understanding",
	},
}

// Client implements both domain/llm.AdapterGenerator and domain/llm.ListingExtractor
// using the Anthropic Go SDK. Construct via New and pass to both interfaces at the
// composition root.
type Client struct {
	api     anthropic.Client
	modelID string
	logger  *slog.Logger
}

// New constructs a Claude LLM Client. apiKey is the Anthropic API key. When modelID
// is empty, DefaultModelID is used.
func New(apiKey string, modelID string, logger *slog.Logger) *Client {
	if modelID == "" {
		modelID = DefaultModelID
	}
	return &Client{
		api:     anthropic.NewClient(option.WithAPIKey(apiKey)),
		modelID: modelID,
		logger:  logger,
	}
}

// GenerateAdapter calls Claude to produce a declarative AdapterSpec from a board URL
// and an example page response. The result is data only — it must be human-reviewed
// and approved before the scraper evaluates it.
func (c *Client) GenerateAdapter(ctx context.Context, boardURL string, exampleResponse string) (*domainllm.AdapterSpec, error) {
	schemaBytes, err := json.Marshal(adapterSpecSchema())
	if err != nil {
		return nil, fmt.Errorf("marshalling adapter schema: %w", err)
	}

	userMsg := fmt.Sprintf("Board URL: %s\n\nExample response:\n%s", boardURL, exampleResponse)

	msg, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.modelID,
		MaxTokens: 4096,
		System: []anthropic.TextBlockParam{
			{
				Text:         adapterSystemPrompt,
				CacheControl: anthropic.CacheControlEphemeralParam{},
			},
		},
		Tools: []anthropic.ToolUnionParam{
			anthropic.ToolUnionParamOfTool(
				anthropic.ToolInputSchemaParam{Properties: json.RawMessage(schemaBytes)},
				"generate_adapter",
			),
		},
		ToolChoice: anthropic.ToolChoiceParamOfTool("generate_adapter"),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(userMsg)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling claude for adapter generation: %w", err)
	}

	raw, err := extractToolInput(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("extracting adapter tool input: %w", err)
	}

	var spec domainllm.AdapterSpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		return nil, fmt.Errorf("unmarshalling adapter spec: %w", err)
	}

	c.logger.InfoContext(ctx, "adapter generated", "board_url", boardURL)
	return &spec, nil
}

// Extract calls Claude to extract structured fields from a raw job listing payload.
// Each returned field carries a per-field confidence score and the listing carries
// an overall understanding score.
//
// The stable system prompt and extraction schema are sent with cache-control breakpoints
// so they are cached across repeated calls, reducing cost on bulk extraction runs.
func (c *Client) Extract(ctx context.Context, raw string) (*domainllm.ExtractedListing, error) {
	schemaBytes, err := json.Marshal(extractionSchema)
	if err != nil {
		return nil, fmt.Errorf("marshalling extraction schema: %w", err)
	}

	msg, err := c.api.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     c.modelID,
		MaxTokens: 2048,
		System: []anthropic.TextBlockParam{
			{
				Text:         extractionSystemPrompt,
				CacheControl: anthropic.CacheControlEphemeralParam{},
			},
		},
		Tools: []anthropic.ToolUnionParam{
			{
				OfTool: &anthropic.ToolParam{
					Name:         "extract_listing",
					Description:  anthropic.String("Extract structured fields from a job listing"),
					InputSchema:  anthropic.ToolInputSchemaParam{Properties: json.RawMessage(schemaBytes)},
					CacheControl: anthropic.CacheControlEphemeralParam{},
				},
			},
		},
		ToolChoice: anthropic.ToolChoiceParamOfTool("extract_listing"),
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(raw)),
		},
	})
	if err != nil {
		return nil, fmt.Errorf("calling claude for extraction: %w", err)
	}

	toolInput, err := extractToolInput(msg.Content)
	if err != nil {
		return nil, fmt.Errorf("extracting listing tool input: %w", err)
	}

	listing, err := parseExtractedListing(toolInput)
	if err != nil {
		return nil, fmt.Errorf("parsing extracted listing: %w", err)
	}

	c.logger.InfoContext(ctx, "listing extracted",
		"understanding", listing.Understanding.Int(),
		"model", c.modelID,
	)
	return listing, nil
}

// extractToolInput finds the first tool_use block in the response and returns its
// raw JSON input.
func extractToolInput(blocks []anthropic.ContentBlockUnion) (json.RawMessage, error) {
	for _, block := range blocks {
		if block.Type == "tool_use" {
			return block.Input, nil
		}
	}
	return nil, fmt.Errorf("no tool_use block in claude response")
}

// rawExtractionResponse is used to decode the flat JSON from Claude before mapping
// to the typed ExtractedListing. This avoids needing generics-aware JSON unmarshalling
// for the intermediate representation.
type rawExtractionResponse struct {
	Skills struct {
		Value      []string `json:"value"`
		Confidence uint8    `json:"confidence"`
	} `json:"skills"`
	RemotePolicy struct {
		Value      string `json:"value"`
		Confidence uint8  `json:"confidence"`
	} `json:"remote_policy"`
	OfficeDays struct {
		Value      int   `json:"value"`
		Confidence uint8 `json:"confidence"`
	} `json:"office_days"`
	ContractType struct {
		Value      string `json:"value"`
		Confidence uint8  `json:"confidence"`
	} `json:"contract_type"`
	WorkingDays struct {
		Value      string `json:"value"`
		Confidence uint8  `json:"confidence"`
	} `json:"working_days"`
	SalaryMin struct {
		Value      *int64 `json:"value"`
		Confidence uint8  `json:"confidence"`
	} `json:"salary_min"`
	SalaryMax struct {
		Value      *int64 `json:"value"`
		Confidence uint8  `json:"confidence"`
	} `json:"salary_max"`
	Seniority struct {
		Value      string `json:"value"`
		Confidence uint8  `json:"confidence"`
	} `json:"seniority"`
	Recruiter struct {
		Value *struct {
			Name        string `json:"name"`
			Email       string `json:"email"`
			LinkedInURL string `json:"linkedin_url"`
			Phone       string `json:"phone"`
		} `json:"value"`
		Confidence uint8 `json:"confidence"`
	} `json:"recruiter"`
	Understanding uint8 `json:"understanding"`
}

// parseExtractedListing maps the raw Claude JSON response to the typed ExtractedListing,
// converting uint8 scores to kernel.Confidence / kernel.Understanding and clamping
// any out-of-range values to 100 (model outputs are usually well-behaved).
func parseExtractedListing(raw json.RawMessage) (*domainllm.ExtractedListing, error) {
	var r rawExtractionResponse
	if err := json.Unmarshal(raw, &r); err != nil {
		return nil, fmt.Errorf("unmarshalling raw extraction response: %w", err)
	}

	clamp := func(v uint8) uint8 {
		if v > 100 {
			return 100
		}
		return v
	}

	conf := func(v uint8) kernel.Confidence { c, _ := kernel.NewConfidence(clamp(v)); return c }

	listing := &domainllm.ExtractedListing{
		Skills: domainllm.ExtractedField[[]string]{
			Value:      r.Skills.Value,
			Confidence: conf(r.Skills.Confidence),
		},
		RemotePolicy: domainllm.ExtractedField[kernel.RemotePolicy]{
			Value:      kernel.RemotePolicy(r.RemotePolicy.Value),
			Confidence: conf(r.RemotePolicy.Confidence),
		},
		OfficeDays: domainllm.ExtractedField[int]{
			Value:      r.OfficeDays.Value,
			Confidence: conf(r.OfficeDays.Confidence),
		},
		ContractType: domainllm.ExtractedField[kernel.ContractType]{
			Value:      kernel.ContractType(r.ContractType.Value),
			Confidence: conf(r.ContractType.Confidence),
		},
		WorkingDays: domainllm.ExtractedField[kernel.WorkingDays]{
			Value:      kernel.WorkingDays(r.WorkingDays.Value),
			Confidence: conf(r.WorkingDays.Confidence),
		},
		SalaryMin: domainllm.ExtractedField[*int64]{
			Value:      r.SalaryMin.Value,
			Confidence: conf(r.SalaryMin.Confidence),
		},
		SalaryMax: domainllm.ExtractedField[*int64]{
			Value:      r.SalaryMax.Value,
			Confidence: conf(r.SalaryMax.Confidence),
		},
		Seniority: domainllm.ExtractedField[kernel.Seniority]{
			Value:      kernel.Seniority(r.Seniority.Value),
			Confidence: conf(r.Seniority.Confidence),
		},
	}

	if r.Recruiter.Value != nil {
		listing.Recruiter = domainllm.ExtractedField[*domainllm.Recruiter]{
			Value: &domainllm.Recruiter{
				Name:        r.Recruiter.Value.Name,
				Email:       r.Recruiter.Value.Email,
				LinkedInURL: r.Recruiter.Value.LinkedInURL,
				Phone:       r.Recruiter.Value.Phone,
			},
			Confidence: conf(r.Recruiter.Confidence),
		}
	} else {
		listing.Recruiter = domainllm.ExtractedField[*domainllm.Recruiter]{
			Value:      nil,
			Confidence: conf(r.Recruiter.Confidence),
		}
	}

	u, _ := kernel.NewUnderstanding(clamp(r.Understanding))
	listing.Understanding = u

	return listing, nil
}

// adapterSpecSchema returns the JSON schema for the AdapterSpec, used as the tool
// input schema for adapter generation.
func adapterSpecSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"board":      map[string]interface{}{"type": "string"},
			"fetch_mode": map[string]interface{}{"type": "string", "enum": []string{"json_api", "html"}},
			"search": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url_template":     map[string]interface{}{"type": "string"},
					"method":           map[string]interface{}{"type": "string", "enum": []string{"GET", "POST"}},
					"body_template":    map[string]interface{}{"type": "string"},
					"param_map":        map[string]interface{}{"type": "object"},
					"pagination":       map[string]interface{}{"type": "object"},
					"result_node_path": map[string]interface{}{"type": "string"},
					"result_fields":    map[string]interface{}{"type": "object"},
				},
				"required": []string{"url_template", "method", "param_map", "pagination", "result_node_path", "result_fields"},
			},
			"listing": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"fetch":       map[string]interface{}{"type": "string", "enum": []string{"detail_page", "use_search_payload"}},
					"raw_capture": map[string]interface{}{"type": "string"},
				},
				"required": []string{"fetch", "raw_capture"},
			},
			"incremental": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"cursor_field":     map[string]interface{}{"type": "string"},
					"overlap_buffer":   map[string]interface{}{"type": "string"},
					"safety_max_pages": map[string]interface{}{"type": "integer"},
				},
				"required": []string{"cursor_field", "overlap_buffer", "safety_max_pages"},
			},
		},
		"required": []string{"board", "fetch_mode", "search", "listing", "incremental"},
	}
}
