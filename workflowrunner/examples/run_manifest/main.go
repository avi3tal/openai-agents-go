package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/workflowrunner"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared/constant"
)

func main() {
	manifestPath, useStdout := resolveConfig()

	data, err := readManifest(manifestPath)
	if err != nil {
		fail(fmt.Errorf("read manifest: %w", err))
	}

	var req workflowrunner.WorkflowRequest
	if err := json.Unmarshal(data, &req); err != nil {
		fail(fmt.Errorf("decode manifest: %w", err))
	}

	if useStdout {
		req.Callback = workflowrunner.CallbackDeclaration{Mode: "stdout"}
		req.Callbacks = nil
	}

	builder := workflowrunner.NewDefaultBuilder()
	registerExampleResources(builder)
	service := workflowrunner.NewRunnerService(builder)

	ctx := context.Background()
	task, err := service.Execute(ctx, req)
	if err != nil {
		fail(fmt.Errorf("execute workflow: %w", err))
	}

	result := task.Await()
	if result.Error != nil {
		fail(fmt.Errorf("run failed: %w", result.Error))
	}

	fmt.Printf("Workflow %s completed.\n", result.Value.WorkflowName)
	if result.Value.FinalOutput != nil {
		fmt.Printf("Final output: %v\n", result.Value.FinalOutput)
	}
	if result.Value.LastResponseID != "" {
		fmt.Printf("Last response id: %s\n", result.Value.LastResponseID)
	}
}

func readManifest(path string) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "workflow manifest runner: %v\n", err)
	os.Exit(1)
}

func resolveConfig() (string, bool) {
	manifestPath := ""
	useStdout := false

	if len(os.Args) > 1 {
		manifestPath = strings.TrimSpace(os.Args[1])
	}
	if manifestPath == "" {
		manifestPath = strings.TrimSpace(os.Getenv("WORKFLOW_MANIFEST"))
	}
	if manifestPath == "" {
		manifestPath = defaultManifestPath()
	}

	if len(os.Args) > 2 {
		useStdout = strings.EqualFold(os.Args[2], "stdout")
	} else if v := strings.TrimSpace(os.Getenv("WORKFLOWRUNNER_STDOUT")); v != "" {
		useStdout = strings.EqualFold(v, "true")
	}
	return manifestPath, useStdout
}

func registerExampleResources(builder *workflowrunner.Builder) {
	builder.WithFunctionTool("get_weather", newGetWeatherTool)
	builder.WithFunctionTool("mock_sensitive_files", newMockSensitiveFilesFunctionTool)
	builder.WithHostedMCPTool("mock_sensitive_files", newMockSensitiveFilesTool)
	builder.WithOutputGuardrail("sensitive_data_check", newSensitiveDataGuardrail)
}

func defaultManifestPath() string {
	cwd, err := os.Getwd()
	if err != nil {
		return "workflowrunner/examples/manifests/basic_hello_world.json"
	}
	candidates := []string{
		"workflowrunner/examples/manifests/basic_hello_world.json",
		"./workflowrunner/examples/manifests/basic_hello_world.json",
	}
	for _, candidate := range candidates {
		path := candidate
		if _, statErr := os.Stat(path); statErr == nil {
			return path
		}
		joined := fmt.Sprintf("%s/%s", strings.TrimRight(cwd, "/"), strings.TrimLeft(candidate, "./"))
		if _, statErr := os.Stat(joined); statErr == nil {
			return joined
		}
	}
	return "workflowrunner/examples/manifests/basic_hello_world.json"
}

type getWeatherArgs struct {
	City string `json:"city"`
}

type getWeatherResult struct {
	City             string `json:"city"`
	TemperatureRange string `json:"temperature_range"`
	Conditions       string `json:"conditions"`
}

func newGetWeatherTool(_ context.Context, decl workflowrunner.ToolDeclaration, _ workflowrunner.ToolFactoryEnv) (agents.Tool, error) {
	description := "Get the current weather information for a specified city."
	if decl.Config != nil {
		if v, ok := decl.Config["description"].(string); ok && strings.TrimSpace(v) != "" {
			description = v
		}
	}

	tool := agents.NewFunctionTool("get_weather", description, func(ctx context.Context, args getWeatherArgs) (getWeatherResult, error) {
		// Demo implementation; replace with live weather lookup in production.
		result := getWeatherResult{
			City:             args.City,
			TemperatureRange: "14-20C",
			Conditions:       "Sunny with wind.",
		}
		return result, nil
	})
	return tool, nil
}

func newMockSensitiveFilesTool(_ context.Context, decl workflowrunner.ToolDeclaration, _ workflowrunner.ToolFactoryEnv) (agents.Tool, error) {
	require := "always"
	if decl.ApprovalFlow != nil && strings.TrimSpace(decl.ApprovalFlow.Require) != "" {
		require = strings.TrimSpace(decl.ApprovalFlow.Require)
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

	label := strings.TrimSpace(decl.Name)
	if label == "" {
		label = "mock_sensitive_files"
	}

	return agents.HostedMCPTool{
		ToolConfig: responses.ToolMcpParam{
			ServerLabel: label,
			ServerURL:   param.NewOpt(serverURL),
			RequireApproval: responses.ToolMcpRequireApprovalUnionParam{
				OfMcpToolApprovalSetting: param.NewOpt(require),
			},
			Type: constant.ValueOf[constant.Mcp](),
		},
		OnApprovalRequest: approvalPrompt,
	}, nil
}

func approvalPrompt(ctx context.Context, req responses.ResponseOutputItemMcpApprovalRequest) (agents.MCPToolApprovalFunctionResult, error) {
	if token := strings.TrimSpace(os.Getenv("WORKFLOWRUNNER_MOCK_APPROVAL")); strings.EqualFold(token, "auto_approve") {
		return agents.MCPToolApprovalFunctionResult{Approve: true}, nil
	}
	fmt.Printf("\nApproval required for tool %s (%s)\nArguments: %s\nApprove? [y/N]: ", req.Name, req.ServerLabel, req.Arguments)
	var input string
	_, err := fmt.Scanln(&input)
	if err != nil && !errors.Is(err, io.EOF) {
		return agents.MCPToolApprovalFunctionResult{}, err
	}
	input = strings.TrimSpace(strings.ToLower(input))
	approve := input == "y" || input == "yes"
	if approve {
		return agents.MCPToolApprovalFunctionResult{Approve: true}, nil
	}
	return agents.MCPToolApprovalFunctionResult{Approve: false, Reason: "User declined"}, nil
}

var phoneNumberPattern = regexp.MustCompile(`\b(\+?\d{1,3}[-.\s]?)?(\(\d{3}\)|\d{3})[-.\s]?\d{3}[-.\s]?\d{4}\b`)

func newSensitiveDataGuardrail(_ context.Context, _ workflowrunner.GuardrailDeclaration) (agents.OutputGuardrail, error) {
	return agents.OutputGuardrail{
		Name: "sensitive_data_check",
		GuardrailFunction: func(_ context.Context, _ *agents.Agent, output any) (agents.GuardrailFunctionOutput, error) {
			reasoning := extractStringField(output, "reasoning")
			response := extractStringField(output, "response")

			reasoningTripwire := phoneNumberPattern.MatchString(reasoning)
			responseTripwire := phoneNumberPattern.MatchString(response)
			triggered := reasoningTripwire || responseTripwire

			info := map[string]any{
				"reasoning_contains_phone": reasoningTripwire,
				"response_contains_phone":  responseTripwire,
			}
			return agents.GuardrailFunctionOutput{
				TripwireTriggered: triggered,
				OutputInfo:        info,
			}, nil
		},
	}, nil
}

func extractStringField(output any, field string) string {
	switch v := output.(type) {
	case map[string]any:
		if raw, ok := v[field]; ok {
			switch val := raw.(type) {
			case string:
				return val
			default:
				bytes, err := json.Marshal(val)
				if err == nil {
					return string(bytes)
				}
			}
		}
	case []any:
		var sb strings.Builder
		for _, item := range v {
			sb.WriteString(extractStringField(item, field))
		}
		return sb.String()
	case string:
		if field == "response" {
			return v
		}
	}
	return ""
}

type mockSensitiveFilesArgs struct {
	Pattern string `json:"pattern"`
}

type mockSensitiveFilesResult struct {
	Approved bool     `json:"approved"`
	Matches  []string `json:"matches"`
}

func newMockSensitiveFilesFunctionTool(_ context.Context, decl workflowrunner.ToolDeclaration, _ workflowrunner.ToolFactoryEnv) (agents.Tool, error) {
	description := "Scan for sensitive files. Requires user approval before revealing matches."
	if decl.Config != nil {
		if v, ok := decl.Config["description"].(string); ok && strings.TrimSpace(v) != "" {
			description = v
		}
	}

	files := []string{
		"vault/secret_plans.txt",
		"logs/access.log",
		"notes/secret_thoughts.md",
	}
	if decl.Config != nil {
		if list := extractStringSlice(decl.Config["files"]); len(list) > 0 {
			files = list
		}
	}

	defaultPattern := "secret"
	if decl.Config != nil {
		if v, ok := decl.Config["default_pattern"].(string); ok && strings.TrimSpace(v) != "" {
			defaultPattern = strings.TrimSpace(v)
		}
	}

	tool := agents.NewFunctionTool("mock_sensitive_files", description, func(ctx context.Context, args mockSensitiveFilesArgs) (mockSensitiveFilesResult, error) {
		pattern := strings.TrimSpace(args.Pattern)
		if pattern == "" {
			pattern = defaultPattern
		}
		matches := filterMatches(files, pattern)
		if len(matches) == 0 {
			return mockSensitiveFilesResult{Approved: true, Matches: matches}, nil
		}
		if shouldAutoApprove() {
			return mockSensitiveFilesResult{Approved: true, Matches: matches}, nil
		}
		if promptApproval(matches) {
			return mockSensitiveFilesResult{Approved: true, Matches: matches}, nil
		}
		return mockSensitiveFilesResult{Approved: false, Matches: nil}, fmt.Errorf("access denied by user")
	})
	return tool, nil
}

func extractStringSlice(value any) []string {
	switch v := value.(type) {
	case []string:
		return append([]string(nil), v...)
	case []any:
		result := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok && strings.TrimSpace(s) != "" {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

func filterMatches(files []string, pattern string) []string {
	if pattern == "" {
		return append([]string(nil), files...)
	}
	lowerPattern := strings.ToLower(pattern)
	result := make([]string, 0, len(files))
	for _, file := range files {
		if strings.Contains(strings.ToLower(file), lowerPattern) {
			result = append(result, file)
		}
	}
	return result
}

func shouldAutoApprove() bool {
	token := strings.TrimSpace(os.Getenv("WORKFLOWRUNNER_MOCK_APPROVAL"))
	return strings.EqualFold(token, "auto_approve")
}

func promptApproval(matches []string) bool {
	fmt.Printf("\nApproval required to reveal files: %s\nApprove? [y/N]: ", strings.Join(matches, ", "))
	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false
	}
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}
