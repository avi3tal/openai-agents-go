package workflowrunner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/asynctask"
	"github.com/nlpodyssey/openai-agents-go/tracing"
)

// RunnerService orchestrates building and executing workflow requests.
type RunnerService struct {
	Builder         *Builder
	CallbackFactory func(ctx context.Context, decl CallbackDeclaration) (CallbackPublisher, error)
	StateStore      ExecutionStateStore
}

// GetExecutionState returns the latest persisted execution state for the given session.
func (s *RunnerService) GetExecutionState(ctx context.Context, sessionID string) (WorkflowExecutionState, bool, error) {
	store := s.StateStore
	if store == nil {
		store = NewInMemoryExecutionStateStore()
		s.StateStore = store
	}
	return store.Load(ctx, sessionID)
}

// ClearExecutionState removes any cached execution state for the session.
func (s *RunnerService) ClearExecutionState(ctx context.Context, sessionID string) error {
	store := s.StateStore
	if store == nil {
		store = NewInMemoryExecutionStateStore()
		s.StateStore = store
	}
	return store.Clear(ctx, sessionID)
}

// PendingApprovals returns the outstanding approval requests for the session.
func (s *RunnerService) PendingApprovals(ctx context.Context, sessionID string) ([]ApprovalRequestState, error) {
	state, ok, err := s.GetExecutionState(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, fmt.Errorf("no execution state for session %q", sessionID)
	}
	approvals := make([]ApprovalRequestState, len(state.PendingApprovals))
	copy(approvals, state.PendingApprovals)
	return approvals, nil
}

// ResolveApproval removes a pending approval request from the execution state.
func (s *RunnerService) ResolveApproval(ctx context.Context, sessionID, approvalID string, approve bool) error {
	store := s.StateStore
	if store == nil {
		store = NewInMemoryExecutionStateStore()
		s.StateStore = store
	}
	state, ok, err := store.Load(ctx, sessionID)
	if err != nil {
		return fmt.Errorf("load execution state: %w", err)
	}
	if !ok {
		return fmt.Errorf("no execution state for session %q", sessionID)
	}
	if len(state.PendingApprovals) == 0 {
		return fmt.Errorf("no pending approvals for session %q", sessionID)
	}
	filtered := state.PendingApprovals[:0]
	removed := false
	for _, approval := range state.PendingApprovals {
		if approval.RequestID == approvalID {
			removed = true
			continue
		}
		filtered = append(filtered, approval)
	}
	if !removed {
		return fmt.Errorf("approval id %q not found in session %q", approvalID, sessionID)
	}
	state.PendingApprovals = append([]ApprovalRequestState(nil), filtered...)
	if len(state.PendingApprovals) == 0 && state.Status == ExecutionStatusWaitingApproval {
		if approve {
			state.Status = ExecutionStatusIdle
		} else {
			state.Status = ExecutionStatusFailed
		}
	}
	state.UpdatedAt = time.Now().UTC()
	return store.Save(ctx, state)
}

// Resume validates that the provided workflow request can resume a pending execution and then runs it.
func (s *RunnerService) Resume(ctx context.Context, req WorkflowRequest) (*asynctask.Task[RunSummary], error) {
	sessionID := strings.TrimSpace(req.Session.SessionID)
	if sessionID == "" {
		return nil, errors.New("session.session_id is required for resume")
	}
	resumeToken := strings.TrimSpace(req.Session.ResumeToken)
	if resumeToken == "" {
		return nil, errors.New("session.resume_token is required for resume")
	}
	store := s.StateStore
	if store == nil {
		store = NewInMemoryExecutionStateStore()
		s.StateStore = store
	}
	state, ok, err := store.Load(ctx, sessionID)
	if err != nil {
		return nil, fmt.Errorf("load execution state: %w", err)
	}
	if !ok {
		return nil, fmt.Errorf("no execution state for session %q", sessionID)
	}
	if strings.TrimSpace(state.ResumeToken) != resumeToken {
		return nil, fmt.Errorf("resume token does not match stored state")
	}
	switch state.Status {
	case ExecutionStatusWaitingApproval, ExecutionStatusFailed:
		// allowed
	default:
		return nil, fmt.Errorf("session status %q cannot be resumed", state.Status)
	}
	if state.WorkflowName != "" && req.Workflow.Name != "" && state.WorkflowName != req.Workflow.Name {
		return nil, fmt.Errorf("workflow name mismatch: stored=%q request=%q", state.WorkflowName, req.Workflow.Name)
	}
	return s.Execute(ctx, req)
}

// RunSummary holds metadata about a completed run.
type RunSummary struct {
	WorkflowName   string           `json:"workflow_name"`
	SessionID      string           `json:"session_id"`
	RunID          string           `json:"run_id"`
	ResumeToken    string           `json:"resume_token"`
	FinalOutput    any              `json:"final_output"`
	NewItems       []agents.RunItem `json:"-"`
	LastResponseID string           `json:"last_response_id"`
	StartedAt      time.Time        `json:"started_at"`
	CompletedAt    *time.Time       `json:"completed_at,omitempty"`
	Error          error            `json:"error,omitempty"`
}

// NewRunnerService constructs a RunnerService with sensible defaults.
func NewRunnerService(builder *Builder) *RunnerService {
	if builder == nil {
		builder = NewDefaultBuilder()
	}
	defaultStore := NewInMemoryExecutionStateStore()
	return &RunnerService{
		Builder: builder,
		CallbackFactory: func(ctx context.Context, decl CallbackDeclaration) (CallbackPublisher, error) {
			switch decl.Mode {
			case "", "http":
				return NewHTTPCallbackPublisher(decl.Target, nil), nil
			case "stdout", "stdout_verbose":
				return StdoutCallbackPublisher{}, nil
			default:
				return nil, fmt.Errorf("unsupported callback mode %q", decl.Mode)
			}
		},
		StateStore: defaultStore,
	}
}

// Execute validates, builds, and runs the workflow asynchronously.
func (s *RunnerService) Execute(ctx context.Context, req WorkflowRequest) (*asynctask.Task[RunSummary], error) {
	if s.Builder == nil {
		return nil, errors.New("RunnerService missing Builder")
	}
	buildResult, err := s.Builder.Build(ctx, req)
	if err != nil {
		return nil, err
	}

	inputItems, err := buildInputItems(req.Inputs)
	if err != nil {
		return nil, fmt.Errorf("convert inputs: %w", err)
	}

	callbackFactory := s.CallbackFactory
	if callbackFactory == nil {
		callbackFactory = func(ctx context.Context, decl CallbackDeclaration) (CallbackPublisher, error) {
			return StdoutCallbackPublisher{}, nil
		}
	}
	callbackDecls := collectCallbackDeclarations(req)
	publishers := make([]CallbackPublisher, 0, len(callbackDecls))
	hasStdout := false
	hasVerboseStdout := false
	hasNonStdout := false
	for i := range callbackDecls {
		decl := callbackDecls[i]
		mode := strings.ToLower(decl.Mode)
		if mode == "stdout" || mode == "stdout_verbose" {
			hasStdout = true
			if mode == "stdout_verbose" {
				hasVerboseStdout = true
			}
		} else {
			hasNonStdout = true
		}
		publisher, err := callbackFactory(ctx, decl)
		if err != nil {
			return nil, fmt.Errorf("create callback publisher[%d]: %w", i, err)
		}
		publishers = append(publishers, publisher)
	}
	var publisher CallbackPublisher = multiCallbackPublisher(publishers)

	stateStore := s.StateStore
	if stateStore == nil {
		stateStore = NewInMemoryExecutionStateStore()
	}
	tracker := newExecutionStateTracker(stateStore, req.Session.SessionID, req.Workflow.Name)
	runID, resumeToken := generateRunIdentifiers(req.Session.SessionID)
	consoleEnabled := hasStdout
	consoleVerbose := hasVerboseStdout
	printer := newConsolePrinter(consoleEnabled, consoleVerbose)
	skipPublishing := hasStdout && !hasNonStdout

	return asynctask.CreateTask(ctx, func(taskCtx context.Context) (RunSummary, error) {
		defer func() {
			if closer, ok := buildResult.Session.(interface{ Close() error }); ok {
				_ = closer.Close()
			}
		}()

		summary := RunSummary{
			WorkflowName: req.Workflow.Name,
			SessionID:    req.Session.SessionID,
			RunID:        runID,
			ResumeToken:  resumeToken,
		}
		traceMetadata := buildResult.TraceMetadata
		if traceMetadata == nil {
			traceMetadata = composeTraceMetadata(req)
		}
		traceID := tracing.GenTraceID()
		buildResult.Runner.Config.TraceID = traceID

		traceErr := tracing.RunTrace(taskCtx, tracing.TraceParams{
			WorkflowName: req.Workflow.Name,
			TraceID:      traceID,
			GroupID:      req.Session.SessionID,
			Metadata:     traceMetadata,
		}, func(ctx context.Context, _ tracing.Trace) error {
			startedAt := time.Now().UTC()
			summary.StartedAt = startedAt
			if err := tracker.OnRunStarted(ctx, runID, resumeToken, req.Query, startedAt); err != nil {
				return err
			}
			printer.OnRunStarted(req.Query)
			startEvent := CallbackEvent{
				Type:      "run.started",
				Timestamp: time.Now().UTC(),
				Payload: map[string]any{
					"workflow":     req.Workflow.Name,
					"session":      req.Session.SessionID,
					"query":        req.Query,
					"run_id":       runID,
					"resume_token": resumeToken,
				},
			}
			if !skipPublishing {
				_ = publisher.Publish(ctx, startEvent)
			}

			var runResult *agents.RunResultStreaming
			if len(inputItems) > 0 {
				runResult, err = buildResult.Runner.RunInputsStreamed(ctx, buildResult.StartingAgent, inputItems)
			} else {
				runResult, err = buildResult.Runner.RunStreamed(ctx, buildResult.StartingAgent, req.Query)
			}
			if err != nil {
				runErr := wrapRunError(err)
				summary.Error = runErr
				completed := time.Now().UTC()
				summary.CompletedAt = &completed
				if !skipPublishing {
					_ = publisher.Publish(ctx, CallbackEvent{
						Type:      "run.failed",
						Timestamp: time.Now().UTC(),
						Payload: map[string]any{
							"error": runErr.Error(),
						},
					})
				}
				_ = tracker.OnRunFailed(ctx, runErr)
				printer.OnRunFailed(runErr)
				return runErr
			}

			streamErr := runResult.StreamEvents(func(ev agents.StreamEvent) error {
				if err := tracker.OnStreamEvent(ctx, ev); err != nil {
					return err
				}
				printer.OnStreamEvent(ev)
				if skipPublishing {
					return nil
				}
				event := CallbackEvent{
					Type:      "run.event",
					Timestamp: time.Now().UTC(),
					Payload:   serializeStreamEvent(ev),
				}
				return publisher.Publish(ctx, event)
			})
			if streamErr != nil {
				streamErr = wrapRunError(streamErr)
				summary.Error = streamErr
				completed := time.Now().UTC()
				summary.CompletedAt = &completed
				if !skipPublishing {
					_ = publisher.Publish(ctx, CallbackEvent{
						Type:      "run.failed",
						Timestamp: time.Now().UTC(),
						Payload: map[string]any{
							"error": streamErr.Error(),
						},
					})
				}
				_ = tracker.OnRunFailed(ctx, streamErr)
				printer.OnRunFailed(streamErr)
				return streamErr
			}

			final := runResult.FinalOutput()
			summary.FinalOutput = final
			summary.NewItems = runResult.NewItems()
			summary.LastResponseID = runResult.LastResponseID()
			completedAt := time.Now().UTC()
			summary.CompletedAt = &completedAt

			completeEvent := CallbackEvent{
				Type:      "run.completed",
				Timestamp: time.Now().UTC(),
				Payload: map[string]any{
					"final_output":     final,
					"last_response_id": runResult.LastResponseID(),
					"run_id":           runID,
					"resume_token":     resumeToken,
				},
			}
			if !skipPublishing {
				_ = publisher.Publish(ctx, completeEvent)
			}
			_ = tracker.OnRunCompleted(ctx, runResult.LastResponseID(), final)
			printer.OnRunCompleted(final, displayAgentName(runResult.LastAgent()))
			return nil
		})

		if traceErr != nil && summary.Error == nil {
			summary.Error = traceErr
			_ = tracker.OnRunFailed(taskCtx, traceErr)
			printer.OnRunFailed(traceErr)
			if summary.CompletedAt == nil {
				completed := time.Now().UTC()
				summary.CompletedAt = &completed
			}
		}

		return summary, summary.Error
	}), nil
}

func generateRunIdentifiers(sessionID string) (string, string) {
	runID := tracing.GenTraceID()
	resumeToken := runID
	if strings.TrimSpace(sessionID) != "" {
		resumeToken = fmt.Sprintf("%s:%s", sessionID, runID)
	}
	return runID, resumeToken
}

func wrapRunError(err error) error {
	var agentsErr *agents.AgentsError
	if errors.As(err, &agentsErr) && agentsErr.RunData != nil {
		if agentsErr.RunData.LastAgent != nil {
			return fmt.Errorf("%w (last agent: %s)", err, agentsErr.RunData.LastAgent.Name)
		}
	}
	return err
}

func serializeStreamEvent(event agents.StreamEvent) map[string]any {
	switch ev := event.(type) {
	case agents.RawResponsesStreamEvent:
		payload := map[string]any{
			"event_kind": "raw",
			"type":       ev.Data.Type,
		}
		if raw, err := json.Marshal(ev.Data); err == nil {
			payload["data"] = json.RawMessage(raw)
		} else {
			payload["marshal_error"] = err.Error()
		}
		return payload
	case agents.AgentUpdatedStreamEvent:
		agentName := ""
		if ev.NewAgent != nil {
			agentName = ev.NewAgent.Name
		}
		return map[string]any{
			"event_kind": "agent_updated",
			"agent_name": agentName,
		}
	case agents.RunItemStreamEvent:
		return map[string]any{
			"event_kind": "run_item",
			"name":       string(ev.Name),
			"item":       summarizeRunItem(ev.Item),
		}
	default:
		return map[string]any{
			"event_kind": "unknown",
			"type":       fmt.Sprintf("%T", event),
		}
	}
}

func summarizeRunItem(item agents.RunItem) map[string]any {
	switch v := item.(type) {
	case agents.MessageOutputItem:
		return map[string]any{
			"type":  v.Type,
			"agent": displayAgentName(v.Agent),
			"text":  agents.ItemHelpers().TextMessageOutput(v),
		}
	case agents.ToolCallItem:
		payload := map[string]any{
			"type":      v.Type,
			"agent":     displayAgentName(v.Agent),
			"tool_call": fmt.Sprintf("%T", v.RawItem),
		}
		switch raw := v.RawItem.(type) {
		case agents.ResponseFunctionToolCall:
			payload["function_name"] = raw.Name
		case agents.ResponseFunctionWebSearch:
			payload["web_search_status"] = raw.Status
		case agents.ResponseFileSearchToolCall:
			payload["file_search_status"] = raw.Status
		}
		return payload
	case agents.ToolCallOutputItem:
		return map[string]any{
			"type":   v.Type,
			"agent":  displayAgentName(v.Agent),
			"output": v.Output,
		}
	case agents.HandoffOutputItem:
		return map[string]any{
			"type":         v.Type,
			"agent":        displayAgentName(v.Agent),
			"source_agent": displayAgentName(v.SourceAgent),
			"target_agent": displayAgentName(v.TargetAgent),
		}
	default:
		return map[string]any{
			"type": fmt.Sprintf("%T", item),
		}
	}
}

type multiCallbackPublisher []CallbackPublisher

func (m multiCallbackPublisher) Publish(ctx context.Context, event CallbackEvent) error {
	var firstErr error
	for _, publisher := range m {
		if publisher == nil {
			continue
		}
		if err := publisher.Publish(ctx, event); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func collectCallbackDeclarations(req WorkflowRequest) []CallbackDeclaration {
	decls := make([]CallbackDeclaration, 0, 1+len(req.Callbacks))
	if !callbackIsEmpty(req.Callback) {
		decls = append(decls, req.Callback)
	}
	decls = append(decls, req.Callbacks...)
	return decls
}

func callbackIsEmpty(cb CallbackDeclaration) bool {
	return strings.TrimSpace(cb.Target) == "" && strings.TrimSpace(cb.Mode) == "" && len(cb.Headers) == 0 && cb.Retry == nil
}
