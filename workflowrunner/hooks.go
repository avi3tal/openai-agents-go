package workflowrunner

import (
	"context"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

type combinedRunHooks []agents.RunHooks

func (c combinedRunHooks) OnLLMStart(ctx context.Context, agent *agents.Agent, systemPrompt param.Opt[string], inputItems []agents.TResponseInputItem) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnLLMStart(ctx, agent, systemPrompt, inputItems); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedRunHooks) OnLLMEnd(ctx context.Context, agent *agents.Agent, response agents.ModelResponse) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnLLMEnd(ctx, agent, response); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedRunHooks) OnAgentStart(ctx context.Context, agent *agents.Agent) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnAgentStart(ctx, agent); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedRunHooks) OnAgentEnd(ctx context.Context, agent *agents.Agent, output any) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnAgentEnd(ctx, agent, output); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedRunHooks) OnHandoff(ctx context.Context, fromAgent, toAgent *agents.Agent) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnHandoff(ctx, fromAgent, toAgent); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedRunHooks) OnToolStart(ctx context.Context, agent *agents.Agent, tool agents.Tool) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnToolStart(ctx, agent, tool); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedRunHooks) OnToolEnd(ctx context.Context, agent *agents.Agent, tool agents.Tool, result any) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnToolEnd(ctx, agent, tool, result); err != nil {
			return err
		}
	}
	return nil
}

type combinedAgentHooks []agents.AgentHooks

func (c combinedAgentHooks) OnStart(ctx context.Context, agent *agents.Agent) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnStart(ctx, agent); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedAgentHooks) OnEnd(ctx context.Context, agent *agents.Agent, output any) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnEnd(ctx, agent, output); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedAgentHooks) OnHandoff(ctx context.Context, agent, source *agents.Agent) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnHandoff(ctx, agent, source); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedAgentHooks) OnToolStart(ctx context.Context, agent *agents.Agent, tool agents.Tool, arguments any) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnToolStart(ctx, agent, tool, arguments); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedAgentHooks) OnToolEnd(ctx context.Context, agent *agents.Agent, tool agents.Tool, result any) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnToolEnd(ctx, agent, tool, result); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedAgentHooks) OnLLMStart(ctx context.Context, agent *agents.Agent, systemPrompt param.Opt[string], inputItems []agents.TResponseInputItem) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnLLMStart(ctx, agent, systemPrompt, inputItems); err != nil {
			return err
		}
	}
	return nil
}

func (c combinedAgentHooks) OnLLMEnd(ctx context.Context, agent *agents.Agent, response agents.ModelResponse) error {
	for _, hook := range c {
		if hook == nil {
			continue
		}
		if err := hook.OnLLMEnd(ctx, agent, response); err != nil {
			return err
		}
	}
	return nil
}
