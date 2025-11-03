package workflowrunner

import (
	"errors"
	"fmt"
	"strings"
)

// ValidateWorkflowRequest performs structural validation and returns an error
// describing the first issue encountered.
func ValidateWorkflowRequest(req WorkflowRequest) error {
	if version := strings.TrimSpace(req.Version); version != "" && version != CurrentWorkflowVersion {
		return fmt.Errorf("version %q not supported", req.Version)
	}
	if strings.TrimSpace(req.Query) == "" {
		return errors.New("query is required")
	}
	if err := validateInputs(req.Inputs); err != nil {
		return fmt.Errorf("inputs invalid: %w", err)
	}
	if err := validateSession(req.Session); err != nil {
		return fmt.Errorf("session invalid: %w", err)
	}
	callbackCount := 0
	if !isCallbackEmpty(req.Callback) {
		if err := req.Callback.Validate(); err != nil {
			return fmt.Errorf("callback invalid: %w", err)
		}
		callbackCount++
	}
	for i, cb := range req.Callbacks {
		if err := cb.Validate(); err != nil {
			return fmt.Errorf("callbacks[%d] invalid: %w", i, err)
		}
		callbackCount++
	}
	if callbackCount == 0 {
		return errors.New("at least one callback is required")
	}
	if err := validateWorkflowDeclaration(req.Workflow); err != nil {
		return fmt.Errorf("workflow invalid: %w", err)
	}
	return nil
}

func validateInputs(inputs []WorkflowInput) error {
	for i, input := range inputs {
		if strings.TrimSpace(input.Type) == "" {
			return fmt.Errorf("inputs[%d] missing type", i)
		}
		typeLower := strings.ToLower(input.Type)
		switch typeLower {
		case "text", "message", "json", "image", "audio", "video":
		default:
			return fmt.Errorf("inputs[%d] type %q not supported", i, input.Type)
		}
		if strings.TrimSpace(input.URI) == "" && input.Content == nil {
			return fmt.Errorf("inputs[%d] must provide either uri or content", i)
		}
	}
	return nil
}

func validateSession(session SessionDeclaration) error {
	if session.SessionID == "" {
		return errors.New("session_id is required")
	}
	if session.Credentials.UserID == "" {
		return errors.New("credentials.user_id is required")
	}
	if session.Credentials.AccountID == "" {
		return errors.New("credentials.account_id is required")
	}
	if session.HistorySize < 0 {
		return errors.New("history_size cannot be negative")
	}
	if session.MaxTurns < 0 {
		return errors.New("max_turns cannot be negative")
	}
	if store := strings.TrimSpace(session.PersistentStore); store != "" {
		switch strings.ToLower(store) {
		case "sqlite", "postgres":
		default:
			return fmt.Errorf("persistent_store %q not supported", session.PersistentStore)
		}
	}
	return nil
}

func validateWorkflowDeclaration(workflow WorkflowDeclaration) error {
	if workflow.Name == "" {
		return errors.New("name is required")
	}
	if workflow.StartingAgent == "" {
		return errors.New("starting_agent is required")
	}
	if len(workflow.Agents) == 0 {
		return errors.New("agents cannot be empty")
	}

	seen := make(map[string]struct{}, len(workflow.Agents))
	for i, agent := range workflow.Agents {
		if agent.Name == "" {
			return fmt.Errorf("agents[%d] missing name", i)
		}
		if _, dup := seen[agent.Name]; dup {
			return fmt.Errorf("duplicate agent name %q", agent.Name)
		}
		seen[agent.Name] = struct{}{}
		if err := validateAgentDeclaration(agent); err != nil {
			return fmt.Errorf("agent %q invalid: %w", agent.Name, err)
		}
	}

	for i, hook := range workflow.OnStart {
		if strings.TrimSpace(hook) == "" {
			return fmt.Errorf("on_start[%d] cannot be empty", i)
		}
	}
	for i, hook := range workflow.OnFinish {
		if strings.TrimSpace(hook) == "" {
			return fmt.Errorf("on_finish[%d] cannot be empty", i)
		}
	}
	for i, hook := range workflow.OnError {
		if strings.TrimSpace(hook) == "" {
			return fmt.Errorf("on_error[%d] cannot be empty", i)
		}
	}

	if _, ok := seen[workflow.StartingAgent]; !ok {
		return fmt.Errorf("starting_agent %q not found in agents", workflow.StartingAgent)
	}
	for _, agent := range workflow.Agents {
		for _, h := range agent.Handoffs {
			if strings.TrimSpace(h.Agent) == "" {
				return fmt.Errorf("agent %q handoff missing agent", agent.Name)
			}
			if _, ok := seen[h.Agent]; !ok {
				return fmt.Errorf("agent %q handoff %q not found", agent.Name, h.Agent)
			}
		}
		for _, tool := range agent.AgentTools {
			if _, ok := seen[tool.AgentName]; !ok {
				return fmt.Errorf("agent %q agent_tool references unknown agent %q", agent.Name, tool.AgentName)
			}
		}
	}
	return nil
}

func validateAgentDeclaration(agent AgentDeclaration) error {
	if agent.Model != nil {
		if agent.Model.Model == "" {
			return errors.New("model.model is required when model is present")
		}
	}
	for _, tool := range agent.Tools {
		if strings.TrimSpace(tool.Type) == "" {
			return fmt.Errorf("tool missing type")
		}
		typeLower := strings.ToLower(tool.Type)
		switch typeLower {
		case "function":
			ref := strings.TrimSpace(tool.FunctionRef)
			if ref == "" {
				ref = configString(tool.Config, "function_ref")
			}
			if ref == "" && strings.TrimSpace(tool.Name) == "" {
				return fmt.Errorf("function tool requires function_ref or name")
			}
		case "computer":
			provider := configString(tool.Config, "provider")
			if provider == "" && strings.TrimSpace(tool.Name) == "" {
				return fmt.Errorf("computer tool requires config.provider or name")
			}
		case "local_shell":
			executor := configString(tool.Config, "executor_ref")
			if executor == "" && strings.TrimSpace(tool.Name) == "" {
				return fmt.Errorf("local_shell tool requires config.executor_ref or name")
			}
		}
	}
	for _, tool := range agent.Tools {
		if tool.ApprovalFlow != nil {
			switch strings.ToLower(tool.ApprovalFlow.Require) {
			case "", "never", "always", "sensitive":
			default:
				return fmt.Errorf("tool approval_flow.require %q not supported", tool.ApprovalFlow.Require)
			}
			switch strings.ToLower(tool.ApprovalFlow.ResumeMode) {
			case "", "auto", "manual":
			default:
				return fmt.Errorf("tool approval_flow.resume_mode %q not supported", tool.ApprovalFlow.ResumeMode)
			}
		}
		for i, hook := range tool.Hooks {
			if strings.TrimSpace(hook) == "" {
				return fmt.Errorf("tool hook[%d] cannot be empty", i)
			}
		}
	}
	for _, mcp := range agent.MCPServers {
		if strings.TrimSpace(mcp.Address) == "" {
			return fmt.Errorf("mcp address is required")
		}
	}
	for _, gr := range append(agent.InputGuardrails, agent.OutputGuardrails...) {
		if strings.TrimSpace(gr.Name) == "" {
			return fmt.Errorf("guardrail missing name")
		}
		switch strings.ToLower(gr.Mode) {
		case "", "blocking", "monitor":
		default:
			return fmt.Errorf("guardrail %q mode %q not supported", gr.Name, gr.Mode)
		}
	}
	for i, hook := range agent.Hooks {
		if strings.TrimSpace(hook) == "" {
			return fmt.Errorf("agent hook[%d] cannot be empty", i)
		}
	}
	if agent.ToolUseBehavior != nil {
		mode := strings.TrimSpace(agent.ToolUseBehavior.Mode)
		switch strings.ToLower(mode) {
		case "", "default":
			// ok; default will reset to SDK defaults
		case "run_llm_again", "stop_on_first_tool":
			// nothing additional
		case "stop_at_tools":
			if len(agent.ToolUseBehavior.ToolNames) == 0 {
				return errors.New("tool_use_behavior.stop_at_tools requires tool_names")
			}
		case "custom":
			if strings.TrimSpace(agent.ToolUseBehavior.Handler) == "" {
				return errors.New("tool_use_behavior.custom requires handler")
			}
		default:
			return fmt.Errorf("tool_use_behavior mode %q not supported", agent.ToolUseBehavior.Mode)
		}
	}
	return nil
}

func isCallbackEmpty(cb CallbackDeclaration) bool {
	return strings.TrimSpace(cb.Target) == "" && strings.TrimSpace(cb.Mode) == "" && len(cb.Headers) == 0 && cb.Retry == nil
}

func configString(config map[string]any, key string) string {
	if config == nil {
		return ""
	}
	if raw, ok := config[key]; ok {
		switch v := raw.(type) {
		case string:
			return strings.TrimSpace(v)
		}
	}
	return ""
}
