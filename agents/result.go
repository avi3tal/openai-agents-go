// Copyright 2025 The NLP Odyssey Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package agents

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"
	"sync/atomic"

	"github.com/nlpodyssey/openai-agents-go/asyncqueue"
	"github.com/nlpodyssey/openai-agents-go/asynctask"
	"github.com/nlpodyssey/openai-agents-go/tracing"
)

type RunResult struct {
	// The original input items i.e. the items before Run() was called. This may be a mutated
	// version of the input, if there are handoff input filters that mutate the input.
	Input Input

	// The new items generated during the agent run.
	// These include things like new messages, tool calls and their outputs, etc.
	NewItems []RunItem

	// The raw LLM responses generated by the model during the agent run.
	RawResponses []ModelResponse

	// The output of the last agent.
	FinalOutput any

	// Guardrail results for the input messages.
	InputGuardrailResults []InputGuardrailResult

	// Guardrail results for the final output of the agent.
	OutputGuardrailResults []OutputGuardrailResult

	// The LastAgent that was run.
	LastAgent *Agent
}

func (r RunResult) String() string {
	return PrettyPrintResult(r)
}

// ToInputList creates a new input list, merging the original input with all the new items generated.
func (r RunResult) ToInputList() []TResponseInputItem {
	return toInputList(r.Input, r.NewItems)
}

// LastResponseID is a convenience method to get the response ID of the last model response.
func (r RunResult) LastResponseID() string {
	return lastResponseID(r.RawResponses)
}

// RunResultStreaming is the result of an agent run in streaming mode.
// You can use the `StreamEvents` method to receive semantic events as they are generated.
//
// The streaming method will return the following errors:
// - A MaxTurnsExceededError if the agent exceeds the max_turns limit.
// - A *GuardrailTripwireTriggeredError error if a guardrail is tripped.
type RunResultStreaming struct {
	context                context.Context
	input                  *atomic.Pointer[Input]
	newItems               *atomic.Pointer[[]RunItem]
	rawResponses           *atomic.Pointer[[]ModelResponse]
	finalOutput            *atomic.Value
	inputGuardrailResults  *atomic.Pointer[[]InputGuardrailResult]
	outputGuardrailResults *atomic.Pointer[[]OutputGuardrailResult]
	currentAgent           *atomic.Pointer[Agent]
	currentTurn            *atomic.Uint64
	maxTurns               *atomic.Uint64
	currentAgentOutputType *atomic.Pointer[OutputTypeInterface]
	trace                  *atomic.Pointer[tracing.Trace]
	isComplete             *atomic.Bool
	eventQueue             *asyncqueue.Queue[StreamEvent]
	inputGuardrailQueue    *asyncqueue.Queue[InputGuardrailResult]
	runImplTask            *atomic.Pointer[asynctask.TaskNoValue]
	inputGuardrailsTask    *atomic.Pointer[asynctask.TaskNoValue]
	outputGuardrailsTask   *atomic.Pointer[asynctask.Task[[]OutputGuardrailResult]]
	storedError            *atomic.Pointer[error]
}

func newRunResultStreaming(ctx context.Context) *RunResultStreaming {
	return &RunResultStreaming{
		context:                ctx,
		input:                  newZeroValAtomicPointer[Input](),
		newItems:               newZeroValAtomicPointer[[]RunItem](),
		rawResponses:           newZeroValAtomicPointer[[]ModelResponse](),
		finalOutput:            new(atomic.Value),
		inputGuardrailResults:  newZeroValAtomicPointer[[]InputGuardrailResult](),
		outputGuardrailResults: newZeroValAtomicPointer[[]OutputGuardrailResult](),
		currentAgent:           new(atomic.Pointer[Agent]),
		currentTurn:            new(atomic.Uint64),
		maxTurns:               new(atomic.Uint64),
		currentAgentOutputType: newZeroValAtomicPointer[OutputTypeInterface](),
		trace:                  newZeroValAtomicPointer[tracing.Trace](),
		isComplete:             new(atomic.Bool),
		eventQueue:             asyncqueue.New[StreamEvent](),
		inputGuardrailQueue:    asyncqueue.New[InputGuardrailResult](),
		runImplTask:            new(atomic.Pointer[asynctask.TaskNoValue]),
		inputGuardrailsTask:    new(atomic.Pointer[asynctask.TaskNoValue]),
		outputGuardrailsTask:   new(atomic.Pointer[asynctask.Task[[]OutputGuardrailResult]]),
		storedError:            newZeroValAtomicPointer[error](),
	}
}

func newZeroValAtomicPointer[T any]() *atomic.Pointer[T] {
	var zero T
	p := new(atomic.Pointer[T])
	p.Store(&zero)
	return p
}

// Input returns the original input items i.e. the items before Run() was called.
// This may be a mutated version of the input, if there are handoff input filters that mutate the input.
func (r *RunResultStreaming) Input() Input     { return *r.input.Load() }
func (r *RunResultStreaming) setInput(v Input) { r.input.Store(&v) }

// NewItems returns the new items generated during the agent run.
// These include things like new messages, tool calls and their outputs, etc.
func (r *RunResultStreaming) NewItems() []RunItem     { return *r.newItems.Load() }
func (r *RunResultStreaming) setNewItems(v []RunItem) { r.newItems.Store(&v) }

// RawResponses returns the raw LLM responses generated by the model during the agent run.
func (r *RunResultStreaming) RawResponses() []ModelResponse     { return *r.rawResponses.Load() }
func (r *RunResultStreaming) setRawResponses(v []ModelResponse) { r.rawResponses.Store(&v) }
func (r *RunResultStreaming) appendRawResponses(v ...ModelResponse) {
	r.setRawResponses(append(r.RawResponses(), v...))
}

// FinalOutput returns the output of the last agent.
// This is nil until the agent has finished running.
func (r *RunResultStreaming) FinalOutput() any     { return r.finalOutput.Load() }
func (r *RunResultStreaming) setFinalOutput(v any) { r.finalOutput.Store(v) }

// InputGuardrailResults returns the guardrail results for the input messages.
func (r *RunResultStreaming) InputGuardrailResults() []InputGuardrailResult {
	return *r.inputGuardrailResults.Load()
}
func (r *RunResultStreaming) setInputGuardrailResults(v []InputGuardrailResult) {
	r.inputGuardrailResults.Store(&v)
}

// OutputGuardrailResults returns the guardrail results for the final output of the agent.
func (r *RunResultStreaming) OutputGuardrailResults() []OutputGuardrailResult {
	return *r.outputGuardrailResults.Load()
}
func (r *RunResultStreaming) setOutputGuardrailResults(v []OutputGuardrailResult) {
	r.outputGuardrailResults.Store(&v)
}

// CurrentAgent returns the current agent that is running.
func (r *RunResultStreaming) CurrentAgent() *Agent     { return r.currentAgent.Load() }
func (r *RunResultStreaming) setCurrentAgent(v *Agent) { r.currentAgent.Store(v) }

// CurrentTurn returns the current turn number.
func (r *RunResultStreaming) CurrentTurn() uint64     { return r.currentTurn.Load() }
func (r *RunResultStreaming) setCurrentTurn(v uint64) { r.currentTurn.Store(v) }

// MaxTurns returns the maximum number of turns the agent can run for.
func (r *RunResultStreaming) MaxTurns() uint64     { return r.maxTurns.Load() }
func (r *RunResultStreaming) setMaxTurns(v uint64) { r.maxTurns.Store(v) }

func (r *RunResultStreaming) getCurrentAgentOutputType() OutputTypeInterface {
	return *r.currentAgentOutputType.Load()
}
func (r *RunResultStreaming) setCurrentAgentOutputType(v OutputTypeInterface) {
	r.currentAgentOutputType.Store(&v)
}

func (r *RunResultStreaming) getTrace() tracing.Trace {
	return *r.trace.Load()
}
func (r *RunResultStreaming) setTrace(v tracing.Trace) {
	r.trace.Store(&v)
}

// IsComplete reports whether the agent has finished running.
func (r *RunResultStreaming) IsComplete() bool     { return r.isComplete.Load() }
func (r *RunResultStreaming) setIsComplete(v bool) { r.isComplete.Store(v) }
func (r *RunResultStreaming) markAsComplete()      { r.setIsComplete(true) }

func (r *RunResultStreaming) getRunImplTask() *asynctask.TaskNoValue  { return r.runImplTask.Load() }
func (r *RunResultStreaming) setRunImplTask(v *asynctask.TaskNoValue) { r.runImplTask.Store(v) }
func (r *RunResultStreaming) createRunImplTask(ctx context.Context, fn func(context.Context) error) {
	r.setRunImplTask(asynctask.CreateTaskNoValue(ctx, fn))
}

func (r *RunResultStreaming) getInputGuardrailsTask() *asynctask.TaskNoValue {
	return r.inputGuardrailsTask.Load()
}
func (r *RunResultStreaming) setInputGuardrailsTask(v *asynctask.TaskNoValue) {
	r.inputGuardrailsTask.Store(v)
}
func (r *RunResultStreaming) createInputGuardrailsTask(ctx context.Context, fn func(context.Context) error) {
	r.setInputGuardrailsTask(asynctask.CreateTaskNoValue(ctx, fn))
}

func (r *RunResultStreaming) getOutputGuardrailsTask() *asynctask.Task[[]OutputGuardrailResult] {
	return r.outputGuardrailsTask.Load()
}
func (r *RunResultStreaming) setOutputGuardrailsTask(v *asynctask.Task[[]OutputGuardrailResult]) {
	r.outputGuardrailsTask.Store(v)
}
func (r *RunResultStreaming) createOutputGuardrailsTask(ctx context.Context, fn func(context.Context) ([]OutputGuardrailResult, error)) {
	r.setOutputGuardrailsTask(asynctask.CreateTask(ctx, fn))
}

func (r *RunResultStreaming) getStoredError() error  { return *r.storedError.Load() }
func (r *RunResultStreaming) setStoredError(v error) { r.storedError.Store(&v) }

// ToInputList creates a new input list, merging the original input with all the new items generated.
func (r *RunResultStreaming) ToInputList() []TResponseInputItem {
	return toInputList(r.Input(), r.NewItems())
}

// LastResponseID is a convenience method to get the response ID of the last model response.
func (r *RunResultStreaming) LastResponseID() string {
	return lastResponseID(r.RawResponses())
}

// The LastAgent that was run.
// Updates as the agent run progresses, so the true last agent is only
// available after the agent run is complete.
func (r *RunResultStreaming) LastAgent() *Agent {
	return r.CurrentAgent()
}

// Cancel the streaming run, stopping all background tasks and marking the run as complete.
func (r *RunResultStreaming) Cancel() {
	r.markAsComplete() // Mark the run as complete to stop event streaming
	r.cleanupTasks()   // Cancel all running tasks
	r.awaitTasks()

	// Optionally, clear the event queue to prevent processing stale events
	for !r.eventQueue.IsEmpty() {
		_, _ = r.eventQueue.GetNoWait()
	}
	for !r.inputGuardrailQueue.IsEmpty() {
		_, _ = r.inputGuardrailQueue.GetNoWait()
	}
}

// StreamEvents streams deltas for new items as they are generated.
// We're using the types from the OpenAI Responses API, so these are semantic events:
// each event has a `Type` field that describes the type of the event, along with the data for that event.
//
// Possible well-known errors returned:
//   - A MaxTurnsExceededError if the agent exceeds the MaxTurns limit.
//   - A *GuardrailTripwireTriggeredError if a guardrail is tripped.
func (r *RunResultStreaming) StreamEvents(fn func(StreamEvent) error) error {
	for {
		err := r.checkErrors()
		if err != nil {
			return err
		}

		if r.getStoredError() != nil {
			Logger().Debug("Breaking due to stored error")
			r.markAsComplete()
			break
		}

		if r.IsComplete() && r.eventQueue.IsEmpty() {
			break
		}

		item := r.eventQueue.Get()

		if _, ok := item.(queueCompleteSentinel); ok {
			// Check for errors, in case the queue was completed due to an error
			if err = r.checkErrors(); err != nil {
				return err
			}
			break
		}

		err = fn(item)
		if err != nil {
			return err
		}
	}

	r.awaitTasks()
	if err := r.checkErrors(); err != nil {
		return err
	}

	r.cleanupTasks()
	return r.getStoredError()
}

// createErrorDetails returns a RunErrorDetails object considering the current attributes of the object.
func (r *RunResultStreaming) createErrorDetails() *RunErrorDetails {
	return &RunErrorDetails{
		Context:                r.context,
		Input:                  r.Input(),
		NewItems:               r.NewItems(),
		RawResponses:           r.RawResponses(),
		LastAgent:              r.CurrentAgent(),
		InputGuardrailResults:  r.InputGuardrailResults(),
		OutputGuardrailResults: r.OutputGuardrailResults(),
	}
}

func (r *RunResultStreaming) checkErrors() error {
	if r.CurrentTurn() > r.MaxTurns() {
		maxTurnsErr := MaxTurnsExceededErrorf("Max turns (%d) exceeded", r.MaxTurns())
		maxTurnsErr.AgentsError.RunData = r.createErrorDetails()
		r.setStoredError(maxTurnsErr)
	}

	// Fetch all the completed guardrail results from the queue and set an error if needed
	for !r.inputGuardrailQueue.IsEmpty() {
		guardrailResult, ok := r.inputGuardrailQueue.GetNoWait()
		if ok && guardrailResult.Output.TripwireTriggered {
			tripwireErr := NewInputGuardrailTripwireTriggeredError(guardrailResult)
			tripwireErr.AgentsError.RunData = r.createErrorDetails()
			r.setStoredError(tripwireErr)
		}
	}

	// Check the tasks for any error
	if t := r.getRunImplTask(); t != nil && t.IsDone() {
		result := t.Await()
		if err := result.Error; err != nil {
			var agentsErr *AgentsError
			if errors.As(err, &agentsErr) && agentsErr.RunData == nil {
				agentsErr.RunData = r.createErrorDetails()
			}
			r.setStoredError(fmt.Errorf("run-impl task error: %w", err))
		}
	}

	if t := r.getInputGuardrailsTask(); t != nil && t.IsDone() {
		result := t.Await()
		if err := result.Error; err != nil {
			var agentsErr *AgentsError
			if errors.As(err, &agentsErr) && agentsErr.RunData == nil {
				agentsErr.RunData = r.createErrorDetails()
			}
			r.setStoredError(fmt.Errorf("input guardrails task error: %w", err))
		}
	}

	if t := r.getOutputGuardrailsTask(); t != nil && t.IsDone() {
		result := t.Await()
		if err := result.Error; err != nil {
			var agentsErr *AgentsError
			if errors.As(err, &agentsErr) && agentsErr.RunData == nil {
				agentsErr.RunData = r.createErrorDetails()
			}
			r.setStoredError(fmt.Errorf("output guardrails task error: %w", err))
		}
	}

	return nil
}

func (r *RunResultStreaming) awaitTasks() {
	var wg sync.WaitGroup
	if t := r.getRunImplTask(); t != nil && !t.IsDone() {
		wg.Add(1)
		go func() {
			t.Await()
			wg.Done()
		}()
	}
	if t := r.getInputGuardrailsTask(); t != nil && !t.IsDone() {
		wg.Add(1)
		go func() {
			t.Await()
			wg.Done()
		}()
	}
	if t := r.getOutputGuardrailsTask(); t != nil && !t.IsDone() {
		wg.Add(1)
		go func() {
			t.Await()
			wg.Done()
		}()
	}
	wg.Wait()
}

func (r *RunResultStreaming) cleanupTasks() {
	if t := r.getRunImplTask(); t != nil && !t.IsDone() {
		t.Cancel()
	}
	if t := r.getInputGuardrailsTask(); t != nil && !t.IsDone() {
		t.Cancel()
	}
	if t := r.getOutputGuardrailsTask(); t != nil && !t.IsDone() {
		t.Cancel()
	}
}

func (r *RunResultStreaming) String() string {
	return PrettyPrintRunResultStreaming(*r)
}

func toInputList(input Input, newRunItems []RunItem) []TResponseInputItem {
	originalItems := ItemHelpers().InputToNewInputList(input)

	result := make([]TResponseInputItem, len(newRunItems))
	for i, item := range newRunItems {
		result[i] = item.ToInputItem()
	}

	return slices.Concat(originalItems, result)
}

func lastResponseID(rawResponses []ModelResponse) string {
	if len(rawResponses) == 0 {
		return ""
	}
	return rawResponses[len(rawResponses)-1].ResponseID
}
