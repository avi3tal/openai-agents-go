package workflowrunner

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
	"strings"
	"text/template"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/memory"
	"github.com/nlpodyssey/openai-agents-go/modelsettings"
	"github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared/constant"
)

// ToolFactory creates an agents.Tool from the declaration.
type ToolFactory func(ctx context.Context, decl ToolDeclaration, env ToolFactoryEnv) (agents.Tool, error)

// FunctionToolFactory creates a function-backed tool from a registry entry.
type FunctionToolFactory func(ctx context.Context, decl ToolDeclaration, env ToolFactoryEnv) (agents.Tool, error)

// AgentToolExtractor extracts output from agent-as-tool runs.
type AgentToolExtractor func(ctx context.Context, result agents.RunResult) (string, error)

// InputGuardrailFactory builds input guardrails from declarative definitions.
type InputGuardrailFactory func(ctx context.Context, decl GuardrailDeclaration) (agents.InputGuardrail, error)

// OutputGuardrailFactory builds output guardrails from declarative definitions.
type OutputGuardrailFactory func(ctx context.Context, decl GuardrailDeclaration) (agents.OutputGuardrail, error)

// ToolFactoryEnv provides context when constructing tools.
type ToolFactoryEnv struct {
	AgentName       string
	WorkflowName    string
	RequestMetadata map[string]any
}

// OutputTypeFactory produces custom output type implementations.
type OutputTypeFactory func(ctx context.Context, decl OutputTypeDeclaration) (agents.OutputTypeInterface, error)

// SessionFactory allocates or loads a conversational session.
type SessionFactory func(ctx context.Context, decl SessionDeclaration) (memory.Session, error)

// BuildResult contains the artifacts required to execute a workflow.
type BuildResult struct {
	StartingAgent *agents.Agent
	AgentMap      map[string]*agents.Agent
	Runner        agents.Runner
	Session       memory.Session
	WorkflowName  string
	TraceMetadata map[string]any
}

// Builder converts declarative workflow payloads into executable SDK primitives.
type Builder struct {
	ToolFactories            map[string]ToolFactory
	OutputTypeFactories      map[string]OutputTypeFactory
	SessionFactory           SessionFactory
	SessionFactories         map[string]SessionFactory
	DefaultSessionStore      string
	FunctionToolFactories    map[string]FunctionToolFactory
	ComputerToolFactories    map[string]ToolFactory
	LocalShellExecutors      map[string]agents.LocalShellExecutor
	AgentToolExtractors      map[string]AgentToolExtractor
	AgentHooks               map[string]agents.AgentHooks
	RunHooks                 map[string]agents.RunHooks
	InputGuardrailFactories  map[string]InputGuardrailFactory
	OutputGuardrailFactories map[string]OutputGuardrailFactory
}

// WithFunctionTool registers a function tool factory under the provided reference name.
func (b *Builder) WithFunctionTool(name string, factory FunctionToolFactory) *Builder {
	if b.FunctionToolFactories == nil {
		b.FunctionToolFactories = make(map[string]FunctionToolFactory)
	}
	b.FunctionToolFactories[name] = factory
	return b
}

// WithComputerTool registers a computer tool factory under the provided provider name.
func (b *Builder) WithComputerTool(name string, factory ToolFactory) *Builder {
	if b.ComputerToolFactories == nil {
		b.ComputerToolFactories = make(map[string]ToolFactory)
	}
	b.ComputerToolFactories[name] = factory
	return b
}

// WithLocalShellExecutor registers a local shell executor under the provided reference name.
func (b *Builder) WithLocalShellExecutor(name string, executor agents.LocalShellExecutor) *Builder {
	if b.LocalShellExecutors == nil {
		b.LocalShellExecutors = make(map[string]agents.LocalShellExecutor)
	}
	b.LocalShellExecutors[name] = executor
	return b
}

// WithHostedMCPTool registers a hosted MCP tool factory by label.
func (b *Builder) WithHostedMCPTool(name string, factory ToolFactory) *Builder {
	if b.ToolFactories == nil {
		b.ToolFactories = make(map[string]ToolFactory)
	}
	b.ToolFactories[strings.ToLower(name)] = factory
	return b
}

// WithAgentToolExtractor registers an extractor that can be referenced by agent_tools output_extractor.
func (b *Builder) WithAgentToolExtractor(name string, extractor AgentToolExtractor) *Builder {
	if b.AgentToolExtractors == nil {
		b.AgentToolExtractors = make(map[string]AgentToolExtractor)
	}
	b.AgentToolExtractors[name] = extractor
	return b
}

// WithAgentHooks registers a reusable agent hooks implementation accessible via agent declarations.
func (b *Builder) WithAgentHooks(name string, hooks agents.AgentHooks) *Builder {
	if b.AgentHooks == nil {
		b.AgentHooks = make(map[string]agents.AgentHooks)
	}
	b.AgentHooks[name] = hooks
	return b
}

// WithRunHooks registers run-level hooks referenced by workflow declarations.
func (b *Builder) WithRunHooks(name string, hooks agents.RunHooks) *Builder {
	if b.RunHooks == nil {
		b.RunHooks = make(map[string]agents.RunHooks)
	}
	b.RunHooks[name] = hooks
	return b
}

// WithInputGuardrail registers or overrides an input guardrail factory.
func (b *Builder) WithInputGuardrail(name string, factory InputGuardrailFactory) *Builder {
	if b.InputGuardrailFactories == nil {
		b.InputGuardrailFactories = make(map[string]InputGuardrailFactory)
	}
	b.InputGuardrailFactories[strings.ToLower(name)] = factory
	return b
}

// WithOutputGuardrail registers or overrides an output guardrail factory.
func (b *Builder) WithOutputGuardrail(name string, factory OutputGuardrailFactory) *Builder {
	if b.OutputGuardrailFactories == nil {
		b.OutputGuardrailFactories = make(map[string]OutputGuardrailFactory)
	}
	b.OutputGuardrailFactories[strings.ToLower(name)] = factory
	return b
}

// NewDefaultBuilder returns a Builder with the builtin registries initialized.
func NewDefaultBuilder() *Builder {
	builder := &Builder{
		OutputTypeFactories: map[string]OutputTypeFactory{
			"json_object": newJSONMapOutputType,
		},
		SessionFactories: map[string]SessionFactory{
			"sqlite":   NewSQLiteSessionFactory("workflowrunner_sessions"),
			"postgres": NewPostgresSessionFactory(""),
		},
		DefaultSessionStore:      "sqlite",
		FunctionToolFactories:    make(map[string]FunctionToolFactory),
		ComputerToolFactories:    make(map[string]ToolFactory),
		LocalShellExecutors:      make(map[string]agents.LocalShellExecutor),
		AgentToolExtractors:      make(map[string]AgentToolExtractor),
		AgentHooks:               make(map[string]agents.AgentHooks),
		RunHooks:                 make(map[string]agents.RunHooks),
		InputGuardrailFactories:  maps.Clone(defaultInputGuardrailFactories),
		OutputGuardrailFactories: maps.Clone(defaultOutputGuardrailFactories),
	}
	builder.ToolFactories = map[string]ToolFactory{
		"web_search":       newWebSearchTool,
		"code_interpreter": newCodeInterpreterTool,
		"file_search":      newFileSearchTool,
		"image_generation": newImageGenerationTool,
		"hosted_mcp":       newHostedMCPTool,
		"function":         builder.buildFunctionTool,
		"computer":         builder.buildComputerTool,
		"local_shell":      builder.buildLocalShellTool,
	}
	builder.WithHostedMCPTool("mock_sensitive_files", builder.buildMockSensitiveFilesTool)
	return builder
}

// Build constructs agents, run configuration, and session resources from the request.
func (b *Builder) Build(ctx context.Context, req WorkflowRequest) (*BuildResult, error) {
	if err := ValidateWorkflowRequest(req); err != nil {
		return nil, err
	}

	var session memory.Session
	useSession := len(req.Inputs) == 0
	if useSession {
		sessionFactory, err := b.resolveSessionFactory(req.Session)
		if err != nil {
			return nil, err
		}
		acquiredSession, err := sessionFactory(ctx, req.Session)
		if err != nil {
			return nil, fmt.Errorf("create session: %w", err)
		}
		session = acquiredSession
	}

	agentMap := make(map[string]*agents.Agent, len(req.Workflow.Agents))
	type pendingConfig struct {
		decl       AgentDeclaration
		agent      *agents.Agent
		agentTools []AgentToolReference
		toolDecls  []ToolDeclaration
	}
	pending := make([]pendingConfig, 0, len(req.Workflow.Agents))

	for _, decl := range req.Workflow.Agents {
		agent := agents.New(decl.Name)
		if decl.DisplayName != "" {
			agent.Name = decl.DisplayName
		}
		if decl.HandoffDescription != "" {
			agent.WithHandoffDescription(decl.HandoffDescription)
		}
		if !decl.Instructions.IsZero() {
			instructions, err := renderInstructions(req, decl)
			if err != nil {
				return nil, fmt.Errorf("agent %q instructions: %w", decl.Name, err)
			}
			if strings.TrimSpace(instructions) != "" {
				agent.WithInstructions(instructions)
			}
		}
		if decl.PromptID != "" {
			agent.WithPrompt(agents.Prompt{ID: decl.PromptID})
		}
		if decl.Model != nil {
			modelSettings, err := applyModelDeclaration(*decl.Model)
			if err != nil {
				return nil, fmt.Errorf("agent %q model: %w", decl.Name, err)
			}
			agent.WithModel(decl.Model.Model)
			agent.WithModelSettings(*modelSettings)
		}
		if decl.OutputType != nil {
			outputType, err := b.buildOutputType(ctx, *decl.OutputType)
			if err != nil {
				return nil, fmt.Errorf("agent %q output type: %w", decl.Name, err)
			}
			agent.WithOutputType(outputType)
		}
		if gr, err := b.buildInputGuardrails(ctx, decl.InputGuardrails); err != nil {
			return nil, fmt.Errorf("agent %q input guardrails: %w", decl.Name, err)
		} else if len(gr) > 0 {
			agent.WithInputGuardrails(gr)
		}
		if gr, err := b.buildOutputGuardrails(ctx, decl.OutputGuardrails); err != nil {
			return nil, fmt.Errorf("agent %q output guardrails: %w", decl.Name, err)
		} else if len(gr) > 0 {
			agent.WithOutputGuardrails(gr)
		}
		if err := applyToolUseBehavior(agent, decl.ToolUseBehavior); err != nil {
			return nil, fmt.Errorf("agent %q tool_use_behavior: %w", decl.Name, err)
		}
		if len(decl.Hooks) > 0 {
			agentHooks, err := b.buildAgentHooks(decl.Hooks)
			if err != nil {
				return nil, fmt.Errorf("agent %q hooks: %w", decl.Name, err)
			}
			if agentHooks != nil {
				agent.WithHooks(agentHooks)
			}
		}
		pending = append(pending, pendingConfig{
			decl:       decl,
			agent:      agent,
			agentTools: decl.AgentTools,
			toolDecls:  append(slices.Clone(decl.Tools), toolsFromMCP(decl.MCPServers)...),
		})
		agentMap[decl.Name] = agent
	}

	// Second pass: attach handoffs and tools.
	// has to be done after all agents are created.
	for _, item := range pending {
		agent := item.agent
		if len(item.decl.Handoffs) > 0 {
			handoffAgents := make([]*agents.Agent, 0, len(item.decl.Handoffs))
			for _, ref := range item.decl.Handoffs {
				target, ok := agentMap[ref.Agent]
				if !ok {
					return nil, fmt.Errorf("agent %q references unknown handoff agent %q", item.decl.Name, ref.Agent)
				}
				handoffAgents = append(handoffAgents, target)
			}
			agent.WithAgentHandoffs(handoffAgents...)
		}
		if len(item.agentTools) > 0 {
			for _, ref := range item.agentTools {
				target, ok := agentMap[ref.AgentName]
				if !ok {
					return nil, fmt.Errorf("agent %q agent_tool references unknown agent %q", item.decl.Name, ref.AgentName)
				}
				params := agents.AgentAsToolParams{
					ToolName:        ref.ToolName,
					ToolDescription: ref.Description,
				}
				if strings.TrimSpace(ref.OutputExtractor) != "" {
					extractor, ok := b.AgentToolExtractors[ref.OutputExtractor]
					if !ok {
						return nil, fmt.Errorf("agent %q agent_tool %q output_extractor %q not registered", item.decl.Name, ref.AgentName, ref.OutputExtractor)
					}
					params.CustomOutputExtractor = extractor
				}
				agent.AddTool(target.AsTool(params))
			}
		}
		if len(item.toolDecls) > 0 {
			for _, toolDecl := range item.toolDecls {
				factory, ok := b.ToolFactories[toolDecl.Type]
				if !ok {
					return nil, fmt.Errorf("agent %q tool type %q not registered", item.decl.Name, toolDecl.Type)
				}
				tool, err := factory(ctx, toolDecl, ToolFactoryEnv{
					AgentName:       item.decl.Name,
					WorkflowName:    req.Workflow.Name,
					RequestMetadata: req.Metadata,
				})
				if err != nil {
					return nil, fmt.Errorf("agent %q tool %q: %w", item.decl.Name, toolDecl.Type, err)
				}
				agent.AddTool(tool)
			}
		}
	}

	startingAgent, ok := agentMap[req.Workflow.StartingAgent]
	if !ok {
		return nil, fmt.Errorf("starting agent %q missing", req.Workflow.StartingAgent)
	}

	runConfig := agents.RunConfig{
		WorkflowName: req.Workflow.Name,
	}
	if req.Session.MaxTurns > 0 {
		runConfig.MaxTurns = uint64(req.Session.MaxTurns)
	}
	hookNames := append([]string{}, req.Workflow.OnStart...)
	hookNames = append(hookNames, req.Workflow.OnFinish...)
	hookNames = append(hookNames, req.Workflow.OnError...)
	if runHooks, err := b.buildRunHooks(hookNames); err != nil {
		return nil, fmt.Errorf("workflow %q hooks: %w", req.Workflow.Name, err)
	} else if runHooks != nil {
		runConfig.Hooks = runHooks
	}
	if session != nil {
		runConfig.Session = session
	}
	if session != nil && req.Session.HistorySize > 0 {
		runConfig.LimitMemory = req.Session.HistorySize
	}
	runConfig.TracingDisabled = false
	runConfig.GroupID = req.Session.SessionID
	traceMetadata := composeTraceMetadata(req)
	runConfig.TraceMetadata = maps.Clone(traceMetadata)

	builderResult := &BuildResult{
		StartingAgent: startingAgent,
		AgentMap:      agentMap,
		Runner:        agents.Runner{Config: runConfig},
		Session:       session,
		WorkflowName:  req.Workflow.Name,
		TraceMetadata: traceMetadata,
	}
	return builderResult, nil
}

func applyModelDeclaration(decl ModelDeclaration) (*modelsettings.ModelSettings, error) {
	if strings.TrimSpace(decl.Provider) != "" && !strings.EqualFold(decl.Provider, "openai") {
		return nil, fmt.Errorf("provider %q not supported (only openai is available in this build)", decl.Provider)
	}
	if strings.TrimSpace(decl.Model) == "" {
		return nil, errors.New("model name cannot be empty")
	}
	settings := modelsettings.ModelSettings{}
	if decl.Temperature != nil {
		settings.Temperature = param.NewOpt(*decl.Temperature)
	}
	if decl.TopP != nil {
		settings.TopP = param.NewOpt(*decl.TopP)
	}
	if decl.MaxTokens != nil {
		settings.MaxTokens = param.NewOpt(*decl.MaxTokens)
	}
	if decl.Verbosity != "" {
		switch strings.ToLower(decl.Verbosity) {
		case "low":
			settings.Verbosity = param.NewOpt(modelsettings.VerbosityLow)
		case "medium":
			settings.Verbosity = param.NewOpt(modelsettings.VerbosityMedium)
		case "high":
			settings.Verbosity = param.NewOpt(modelsettings.VerbosityHigh)
		default:
			return nil, fmt.Errorf("unsupported verbosity %q", decl.Verbosity)
		}
	}
	if decl.Metadata != nil {
		settings.Metadata = decl.Metadata
	}
	if decl.ExtraHeaders != nil {
		settings.ExtraHeaders = decl.ExtraHeaders
	}
	if decl.ExtraQuery != nil {
		settings.ExtraQuery = decl.ExtraQuery
	}
	if decl.Reasoning != nil {
		settings.Reasoning = buildReasoningParam(*decl.Reasoning)
	}
	if strings.TrimSpace(decl.ToolChoice) != "" {
		settings.ToolChoice = modelsettings.ToolChoiceString(decl.ToolChoice)
	}
	return &settings, nil
}

func applyToolUseBehavior(agent *agents.Agent, decl *ToolUseBehaviorDeclaration) error {
	if decl == nil {
		return nil
	}
	mode := strings.ToLower(strings.TrimSpace(decl.Mode))
	switch mode {
	case "", "default":
		return nil
	case "run_llm_again":
		agent.WithToolUseBehavior(agents.RunLLMAgain())
	case "stop_on_first_tool":
		agent.WithToolUseBehavior(agents.StopOnFirstTool())
	case "stop_at_tools":
		agent.WithToolUseBehavior(agents.StopAtTools(decl.ToolNames...))
	case "custom":
		return fmt.Errorf("custom tool_use_behavior handler %q not supported yet", decl.Handler)
	default:
		return fmt.Errorf("unsupported mode %q", decl.Mode)
	}
	return nil
}

func (b *Builder) buildFunctionTool(ctx context.Context, decl ToolDeclaration, env ToolFactoryEnv) (agents.Tool, error) {
	ref := strings.TrimSpace(decl.FunctionRef)
	if ref == "" && decl.Config != nil {
		if value, ok := decl.Config["function_ref"].(string); ok && strings.TrimSpace(value) != "" {
			ref = strings.TrimSpace(value)
		}
	}
	if ref == "" && decl.Name != "" {
		ref = decl.Name
	}
	if ref == "" {
		return nil, errors.New("function tool requires function_ref or name")
	}
	factory, ok := b.FunctionToolFactories[ref]
	if !ok {
		return nil, fmt.Errorf("function tool %q not registered", ref)
	}
	tool, err := factory(ctx, decl, env)
	if err != nil {
		return nil, fmt.Errorf("function tool %q: %w", ref, err)
	}
	if tool == nil {
		return nil, fmt.Errorf("function tool %q factory returned nil", ref)
	}
	return tool, nil
}

func (b *Builder) buildComputerTool(ctx context.Context, decl ToolDeclaration, env ToolFactoryEnv) (agents.Tool, error) {
	var provider string
	if decl.Config != nil {
		if value, ok := decl.Config["provider"].(string); ok {
			provider = strings.TrimSpace(value)
		}
	}
	if provider == "" {
		provider = decl.Name
	}
	if provider == "" {
		return nil, errors.New("computer tool requires config.provider or name")
	}
	factory, ok := b.ComputerToolFactories[provider]
	if !ok {
		return nil, fmt.Errorf("computer tool provider %q not registered", provider)
	}
	tool, err := factory(ctx, decl, env)
	if err != nil {
		return nil, fmt.Errorf("computer tool %q: %w", provider, err)
	}
	if tool == nil {
		return nil, fmt.Errorf("computer tool %q factory returned nil", provider)
	}
	return tool, nil
}

func (b *Builder) buildLocalShellTool(ctx context.Context, decl ToolDeclaration, env ToolFactoryEnv) (agents.Tool, error) {
	var executorRef string
	if decl.Config != nil {
		if value, ok := decl.Config["executor_ref"].(string); ok {
			executorRef = strings.TrimSpace(value)
		}
	}
	if executorRef == "" {
		executorRef = decl.Name
	}
	if executorRef == "" {
		return nil, errors.New("local_shell tool requires config.executor_ref or name")
	}
	executor, ok := b.LocalShellExecutors[executorRef]
	if !ok {
		return nil, fmt.Errorf("local shell executor %q not registered", executorRef)
	}
	return agents.LocalShellTool{Executor: executor}, nil
}

func (b *Builder) buildMockSensitiveFilesTool(ctx context.Context, decl ToolDeclaration, env ToolFactoryEnv) (agents.Tool, error) {
	require := "always"
	if decl.ApprovalFlow != nil && decl.ApprovalFlow.Require != "" {
		require = decl.ApprovalFlow.Require
	} else if decl.Config != nil {
		if v, ok := decl.Config["require_approval"].(string); ok && strings.TrimSpace(v) != "" {
			require = strings.TrimSpace(v)
		}
	}

	serverURL := "mock://sensitive-files"
	if decl.Config != nil {
		if v, ok := decl.Config["server_url"].(string); ok && strings.TrimSpace(v) != "" {
			serverURL = strings.TrimSpace(v)
		}
	}

	tool := agents.HostedMCPTool{
		ToolConfig: responses.ToolMcpParam{
			ServerLabel: "mock_sensitive_files",
			ServerURL:   param.NewOpt(serverURL),
			RequireApproval: responses.ToolMcpRequireApprovalUnionParam{
				OfMcpToolApprovalSetting: param.NewOpt(require),
			},
			Type: constant.ValueOf[constant.Mcp](),
		},
		OnApprovalRequest: func(ctx context.Context, req responses.ResponseOutputItemMcpApprovalRequest) (agents.MCPToolApprovalFunctionResult, error) {
			token := os.Getenv("WORKFLOWRUNNER_MOCK_APPROVAL")
			if strings.EqualFold(token, "auto_approve") {
				return agents.MCPToolApprovalFunctionResult{Approve: true}, nil
			}
			fmt.Printf("\nApproval required for request %s on tool %s (%s)\nArguments: %s\nApprove? [y/N]: ", req.ID, req.Name, req.ServerLabel, req.Arguments)
			var input string
			_, err := fmt.Scanln(&input)
			if err != nil && !errors.Is(err, io.EOF) {
				return agents.MCPToolApprovalFunctionResult{}, err
			}
			input = strings.TrimSpace(strings.ToLower(input))
			approve := input == "y" || input == "yes"
			var reason string
			if !approve {
				reason = "User declined approval"
			}
			return agents.MCPToolApprovalFunctionResult{Approve: approve, Reason: reason}, nil
		},
	}
	return tool, nil
}

func (b *Builder) buildAgentHooks(names []string) (agents.AgentHooks, error) {
	filtered := uniqueNonEmpty(names)
	if len(filtered) == 0 {
		return nil, nil
	}
	hooks := make([]agents.AgentHooks, 0, len(filtered))
	for _, name := range filtered {
		hook, ok := b.AgentHooks[name]
		if !ok {
			return nil, fmt.Errorf("agent hook %q not registered", name)
		}
		if hook != nil {
			hooks = append(hooks, hook)
		}
	}
	if len(hooks) == 0 {
		return nil, nil
	}
	if len(hooks) == 1 {
		return hooks[0], nil
	}
	return combinedAgentHooks(hooks), nil
}

func (b *Builder) buildRunHooks(names []string) (agents.RunHooks, error) {
	filtered := uniqueNonEmpty(names)
	if len(filtered) == 0 {
		return nil, nil
	}
	hooks := make([]agents.RunHooks, 0, len(filtered))
	for _, name := range filtered {
		hook, ok := b.RunHooks[name]
		if !ok {
			return nil, fmt.Errorf("run hook %q not registered", name)
		}
		if hook != nil {
			hooks = append(hooks, hook)
		}
	}
	if len(hooks) == 0 {
		return nil, nil
	}
	if len(hooks) == 1 {
		return hooks[0], nil
	}
	return combinedRunHooks(hooks), nil
}

func buildReasoningParam(decl ReasoningDeclaration) openai.ReasoningParam {
	var result openai.ReasoningParam
	switch strings.ToLower(decl.Effort) {
	case "low":
		result.Effort = openai.ReasoningEffortLow
	case "medium":
		result.Effort = openai.ReasoningEffortMedium
	case "high":
		result.Effort = openai.ReasoningEffortHigh
	case "":
	default:
		result.Effort = openai.ReasoningEffort(decl.Effort)
	}
	switch strings.ToLower(decl.Summary) {
	case "auto":
		result.Summary = openai.ReasoningSummaryAuto
	case "concise":
		result.Summary = openai.ReasoningSummaryConcise
	case "detailed":
		result.Summary = openai.ReasoningSummaryDetailed
	case "":
	default:
		result.Summary = openai.ReasoningSummary(decl.Summary)
	}
	return result
}

func (b *Builder) buildOutputType(ctx context.Context, decl OutputTypeDeclaration) (agents.OutputTypeInterface, error) {
	if decl.Schema == nil {
		factory, ok := b.OutputTypeFactories[decl.Name]
		if !ok {
			return nil, fmt.Errorf("output type %q not registered", decl.Name)
		}
		return factory(ctx, decl)
	}
	name := decl.Name
	if name == "" {
		name = "inline_schema"
	}
	return newSchemaOutputType(name, decl.Strict, decl.Schema)
}

func uniqueNonEmpty(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func toolsFromMCP(decls []MCPDeclaration) []ToolDeclaration {
	if len(decls) == 0 {
		return nil
	}
	out := make([]ToolDeclaration, 0, len(decls))
	for _, d := range decls {
		config := map[string]any{
			"server_label": d.ServerLabel,
			"server_url":   d.Address,
		}
		if d.RequireApproval != "" {
			config["require_approval"] = d.RequireApproval
		}
		for k, v := range d.Additional {
			config[k] = v
		}
		out = append(out, ToolDeclaration{
			Type:   "hosted_mcp",
			Name:   d.ServerLabel,
			Config: config,
		})
	}
	return out
}

func renderInstructions(req WorkflowRequest, decl AgentDeclaration) (string, error) {
	if decl.Instructions.Template == nil {
		return decl.Instructions.Text, nil
	}
	return executeInstructionTemplate(req, decl, *decl.Instructions.Template)
}

func executeInstructionTemplate(req WorkflowRequest, decl AgentDeclaration, tmpl InstructionTemplateDeclaration) (string, error) {
	format := strings.ToLower(strings.TrimSpace(tmpl.Format))
	if format != "" && format != "gotemplate" {
		return "", fmt.Errorf("template format %q not supported", tmpl.Format)
	}
	var buf bytes.Buffer
	t := template.New("instructions")
	if tmpl.Delimiters[0] != "" || tmpl.Delimiters[1] != "" {
		left := tmpl.Delimiters[0]
		right := tmpl.Delimiters[1]
		if left == "" {
			left = "{{"
		}
		if right == "" {
			right = "}}"
		}
		t = t.Delims(left, right)
	}
	parsed, err := t.Parse(tmpl.Template)
	if err != nil {
		return "", fmt.Errorf("parse template: %w", err)
	}
	data := map[string]any{
		"context":  req.Context,
		"metadata": req.Metadata,
		"agent": map[string]any{
			"name":          decl.Name,
			"display_name":  decl.DisplayName,
			"annotations":   decl.Annotations,
			"handoff_names": handoffAgentNames(decl.Handoffs),
		},
		"workflow": map[string]any{
			"name":           req.Workflow.Name,
			"starting_agent": req.Workflow.StartingAgent,
			"metadata":       req.Workflow.Metadata,
			"on_start":       req.Workflow.OnStart,
			"on_finish":      req.Workflow.OnFinish,
			"on_error":       req.Workflow.OnError,
			"agent_names":    workflowAgentNames(req.Workflow.Agents),
		},
		"session": map[string]any{
			"id":               req.Session.SessionID,
			"resume_token":     req.Session.ResumeToken,
			"persistent_store": req.Session.PersistentStore,
			"history_size":     req.Session.HistorySize,
			"max_turns":        req.Session.MaxTurns,
			"credentials": map[string]any{
				"user_id":      req.Session.Credentials.UserID,
				"account_id":   req.Session.Credentials.AccountID,
				"capabilities": req.Session.Credentials.Capabilities,
				"metadata":     req.Session.Credentials.Metadata,
			},
		},
		"request": map[string]any{
			"query":  req.Query,
			"inputs": req.Inputs,
		},
	}
	if len(tmpl.Variables) > 0 {
		userVars := make(map[string]any, len(tmpl.Variables))
		for k, v := range tmpl.Variables {
			userVars[k] = v
		}
		for k, v := range userVars {
			data[k] = v
		}
	}
	if err := parsed.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("execute template: %w", err)
	}
	return buf.String(), nil
}

func workflowAgentNames(decls []AgentDeclaration) []string {
	names := make([]string, 0, len(decls))
	for _, decl := range decls {
		names = append(names, decl.Name)
	}
	return names
}

func handoffAgentNames(decls AgentHandoffDeclarations) []string {
	names := make([]string, 0, len(decls))
	for _, decl := range decls {
		if decl.Agent != "" {
			names = append(names, decl.Agent)
		}
	}
	return names
}

func (b *Builder) resolveSessionFactory(decl SessionDeclaration) (SessionFactory, error) {
	store := strings.ToLower(strings.TrimSpace(decl.PersistentStore))
	if store == "" {
		if b.SessionFactory != nil {
			return b.SessionFactory, nil
		}
		store = b.DefaultSessionStore
		if store == "" {
			store = "sqlite"
		}
	}
	if b.SessionFactories != nil {
		if factory, ok := b.SessionFactories[store]; ok && factory != nil {
			return factory, nil
		}
	}
	switch store {
	case "sqlite":
		return NewSQLiteSessionFactory("workflowrunner_sessions"), nil
	default:
		return nil, fmt.Errorf("persistent_store %q not registered", store)
	}
}
