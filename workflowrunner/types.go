package workflowrunner

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	// CurrentWorkflowVersion marks the latest supported declarative schema version.
	CurrentWorkflowVersion = "v1"
)

// WorkflowRequest represents the top-level payload describing a workflow run.
type WorkflowRequest struct {
	Version   string                `json:"version,omitempty"`
	Query     string                `json:"query"`
	Inputs    []WorkflowInput       `json:"inputs,omitempty"`
	Session   SessionDeclaration    `json:"session"`
	Callback  CallbackDeclaration   `json:"callback"`
	Callbacks []CallbackDeclaration `json:"callbacks,omitempty"`
	Workflow  WorkflowDeclaration   `json:"workflow"`
	Metadata  map[string]any        `json:"metadata,omitempty"`
	Context   map[string]any        `json:"context,omitempty"`
}

// WorkflowInput represents a multimodal input item that can accompany the query.
type WorkflowInput struct {
	Type     string `json:"type"`
	Role     string `json:"role,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	URI      string `json:"uri,omitempty"`
	// Content holds literal payloads (text, JSON objects, base64 blobs).
	Content any `json:"content,omitempty"`
}

// SessionDeclaration carries caller-provided state and execution limits.
type SessionDeclaration struct {
	SessionID       string                `json:"session_id"`
	ResumeToken     string                `json:"resume_token,omitempty"`
	PersistentStore string                `json:"persistent_store,omitempty"`
	StoreConfig     map[string]any        `json:"store_config,omitempty"`
	HistorySize     int                   `json:"history_size,omitempty"`
	MaxTurns        int                   `json:"max_turns,omitempty"`
	Credentials     CredentialDeclaration `json:"credentials"`
}

// CredentialDeclaration contains minimal identity data used for validation / logging.
type CredentialDeclaration struct {
	UserID       string         `json:"user_id"`
	AccountID    string         `json:"account_id"`
	Capabilities []string       `json:"capabilities,omitempty"`
	Metadata     map[string]any `json:"metadata,omitempty"`
}

// CallbackDeclaration describes how streaming events should be published.
type CallbackDeclaration struct {
	Target  string               `json:"target"`
	Mode    string               `json:"mode,omitempty"`
	Headers map[string]string    `json:"headers,omitempty"`
	Retry   *CallbackRetryPolicy `json:"retry,omitempty"`
}

// CallbackRetryPolicy configures HTTP retry behaviour for callbacks.
type CallbackRetryPolicy struct {
	MaxAttempts int     `json:"max_attempts,omitempty"`
	Backoff     float64 `json:"backoff_seconds,omitempty"`
}

// UnmarshalJSON allows callback to be provided as string or object.
func (c *CallbackDeclaration) UnmarshalJSON(data []byte) error {
	type alias CallbackDeclaration
	var (
		asStr string
		asObj alias
	)
	if err := json.Unmarshal(data, &asStr); err == nil {
		c.Target = asStr
		c.Mode = ""
		c.Headers = nil
		c.Retry = nil
		return nil
	}
	if err := json.Unmarshal(data, &asObj); err != nil {
		return err
	}
	*c = CallbackDeclaration(asObj)
	return nil
}

// WorkflowDeclaration defines the agent graph that should be executed.
type WorkflowDeclaration struct {
	Name          string             `json:"name"`
	StartingAgent string             `json:"starting_agent"`
	Agents        []AgentDeclaration `json:"agents"`
	Metadata      map[string]any     `json:"metadata,omitempty"`
	OnStart       []string           `json:"on_start,omitempty"`
	OnFinish      []string           `json:"on_finish,omitempty"`
	OnError       []string           `json:"on_error,omitempty"`
}

// AgentDeclaration captures the configuration of a single agent.
type AgentDeclaration struct {
	Name               string                      `json:"name"`
	DisplayName        string                      `json:"display_name,omitempty"`
	Instructions       InstructionDeclaration      `json:"instructions,omitempty"`
	PromptID           string                      `json:"prompt_id,omitempty"`
	Model              *ModelDeclaration           `json:"model,omitempty"`
	Handoffs           AgentHandoffDeclarations    `json:"handoff,omitempty"`
	AgentTools         []AgentToolReference        `json:"agent_tools,omitempty"`
	Tools              []ToolDeclaration           `json:"tools,omitempty"`
	MCPServers         []MCPDeclaration            `json:"mcp,omitempty"`
	InputGuardrails    []GuardrailDeclaration      `json:"input_guardrails,omitempty"`
	OutputGuardrails   []GuardrailDeclaration      `json:"output_guardrails,omitempty"`
	OutputType         *OutputTypeDeclaration      `json:"output_type,omitempty"`
	ToolUseBehavior    *ToolUseBehaviorDeclaration `json:"tool_use_behavior,omitempty"`
	HandoffDescription string                      `json:"handoff_description,omitempty"`
	Hooks              []string                    `json:"hooks,omitempty"`
	Annotations        map[string]any              `json:"annotations,omitempty"`
}

// AgentToolReference allows referencing another agent as a tool.
type AgentToolReference struct {
	AgentName       string `json:"agent_name"`
	ToolName        string `json:"tool_name,omitempty"`
	Description     string `json:"description,omitempty"`
	OutputExtractor string `json:"output_extractor,omitempty"`
}

// ToolDeclaration represents a tool that should be attached to an agent.
type ToolDeclaration struct {
	Type         string                       `json:"type"`
	Name         string                       `json:"name,omitempty"`
	Config       map[string]any               `json:"config,omitempty"`
	ApprovalFlow *ToolApprovalFlowDeclaration `json:"approval_flow,omitempty"`
	Hooks        []string                     `json:"hooks,omitempty"`
	FunctionRef  string                       `json:"function_ref,omitempty"`
}

// ToolApprovalFlowDeclaration configures human approval expectations for a tool.
type ToolApprovalFlowDeclaration struct {
	Require    string `json:"require,omitempty"`
	ResumeMode string `json:"resume_mode,omitempty"`
}

// MCPDeclaration configures hosted or stdio MCP servers.
type MCPDeclaration struct {
	Type            string         `json:"type,omitempty"`
	ServerLabel     string         `json:"server_label,omitempty"`
	Address         string         `json:"address"`
	RequireApproval string         `json:"require_approval,omitempty"`
	Additional      map[string]any `json:"additional,omitempty"`
}

// GuardrailDeclaration references a reusable guardrail preset.
type GuardrailDeclaration struct {
	Name   string         `json:"name"`
	Config map[string]any `json:"config,omitempty"`
	Target string         `json:"target,omitempty"`
	Mode   string         `json:"mode,omitempty"`
}

// OutputTypeDeclaration describes the expected structured output.
type OutputTypeDeclaration struct {
	Name      string         `json:"name"`
	Strict    bool           `json:"strict,omitempty"`
	Schema    map[string]any `json:"schema,omitempty"`
	PresetRef string         `json:"preset_ref,omitempty"`
}

// ModelDeclaration indicates which model/provider to use and optional settings.
type ModelDeclaration struct {
	Provider          string                `json:"provider,omitempty"`
	Model             string                `json:"model"`
	Temperature       *float64              `json:"temperature,omitempty"`
	TopP              *float64              `json:"top_p,omitempty"`
	MaxTokens         *int64                `json:"max_tokens,omitempty"`
	Reasoning         *ReasoningDeclaration `json:"reasoning,omitempty"`
	Verbosity         string                `json:"verbosity,omitempty"`
	Metadata          map[string]string     `json:"metadata,omitempty"`
	ExtraHeaders      map[string]string     `json:"extra_headers,omitempty"`
	ExtraQuery        map[string]string     `json:"extra_query,omitempty"`
	ToolChoice        string                `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool                 `json:"parallel_tool_calls,omitempty"`
	Truncation        string                `json:"truncation,omitempty"`
}

// ReasoningDeclaration mirrors the subset of OpenAI reasoning parameters we support.
type ReasoningDeclaration struct {
	Effort  string `json:"effort,omitempty"`
	Summary string `json:"summary,omitempty"`
	Tokens  int64  `json:"tokens,omitempty"`
}

// ToolUseBehaviorDeclaration configures how the agent handles tool outputs.
type ToolUseBehaviorDeclaration struct {
	Mode      string   `json:"mode"`
	ToolNames []string `json:"tool_names,omitempty"`
	Handler   string   `json:"handler,omitempty"`
}

// InstructionDeclaration allows plain text or templated instructions.
type InstructionDeclaration struct {
	Text     string                          `json:"-"`
	Template *InstructionTemplateDeclaration `json:"-"`
}

// InstructionTemplateDeclaration describes a templated instruction format.
type InstructionTemplateDeclaration struct {
	Template   string         `json:"template"`
	Format     string         `json:"format,omitempty"`
	Delimiters [2]string      `json:"delimiters,omitempty"`
	Variables  map[string]any `json:"variables,omitempty"`
}

// AgentHandoffDeclaration captures target agent and optional filters.
type AgentHandoffDeclaration struct {
	Agent             string `json:"agent"`
	InputFilter       string `json:"input_filter,omitempty"`
	Instructions      string `json:"instructions,omitempty"`
	InstructionsRef   string `json:"instructions_ref,omitempty"`
	InstructionsScope string `json:"instructions_scope,omitempty"`
}

// AgentHandoffDeclarations is a helper to support string or object syntax.
type AgentHandoffDeclarations []AgentHandoffDeclaration

// UnmarshalJSON accepts either a list of strings or objects for handoffs.
func (h *AgentHandoffDeclarations) UnmarshalJSON(data []byte) error {
	var names []string
	if err := json.Unmarshal(data, &names); err == nil {
		items := make([]AgentHandoffDeclaration, len(names))
		for i, name := range names {
			items[i] = AgentHandoffDeclaration{Agent: name}
		}
		*h = items
		return nil
	}
	var objects []AgentHandoffDeclaration
	if err := json.Unmarshal(data, &objects); err != nil {
		return err
	}
	*h = AgentHandoffDeclarations(objects)
	return nil
}

// MarshalJSON emits strings when the declaration only contains an agent field.
func (h AgentHandoffDeclarations) MarshalJSON() ([]byte, error) {
	allSimple := true
	names := make([]string, len(h))
	for i, item := range h {
		if item.Agent == "" || item.InputFilter != "" || item.Instructions != "" || item.InstructionsRef != "" || item.InstructionsScope != "" {
			allSimple = false
			break
		}
		names[i] = item.Agent
	}
	if allSimple {
		return json.Marshal(names)
	}
	return json.Marshal([]AgentHandoffDeclaration(h))
}

// InstructionText constructs a plain-text instruction declaration.
func InstructionText(text string) InstructionDeclaration {
	return InstructionDeclaration{Text: text}
}

// InstructionTemplate constructs an instruction declaration using a template.
func InstructionTemplate(template string, vars map[string]any) InstructionDeclaration {
	return InstructionDeclaration{
		Template: &InstructionTemplateDeclaration{
			Template:  template,
			Variables: vars,
		},
	}
}

// UnmarshalJSON allows instructions to be provided as string or structured object.
func (i *InstructionDeclaration) UnmarshalJSON(data []byte) error {
	var text string
	if err := json.Unmarshal(data, &text); err == nil {
		i.Text = text
		i.Template = nil
		return nil
	}
	var tmpl InstructionTemplateDeclaration
	if err := json.Unmarshal(data, &tmpl); err != nil {
		return err
	}
	i.Template = &tmpl
	i.Text = ""
	return nil
}

// MarshalJSON prefers emitting the template form when present.
func (i InstructionDeclaration) MarshalJSON() ([]byte, error) {
	if i.Template != nil {
		return json.Marshal(i.Template)
	}
	return json.Marshal(i.Text)
}

// IsZero reports whether no instruction content is provided.
func (i InstructionDeclaration) IsZero() bool {
	return i.Text == "" && i.Template == nil
}

// Validate performs shallow validation of the callback declaration.
func (c *CallbackDeclaration) Validate() error {
	mode := strings.ToLower(c.Mode)
	if mode == "stdout" || mode == "stdout_verbose" {
		return nil
	}
	if strings.TrimSpace(c.Target) == "" {
		return fmt.Errorf("callback target is required")
	}
	if _, err := url.ParseRequestURI(c.Target); err != nil {
		return fmt.Errorf("callback target %q is not a valid URL: %w", c.Target, err)
	}
	return nil
}
