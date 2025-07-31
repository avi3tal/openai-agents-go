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

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/modelsettings"
	"github.com/nlpodyssey/openai-agents-go/tracing"
	"github.com/openai/openai-go/packages/param"
)

/*
Anthropic Claude Provider Example

This example demonstrates how to use Anthropic's Claude models with the OpenAI SDK compatibility layer.
Anthropic provides OpenAI-compatible endpoints that allow you to use Claude models with minimal code changes.

Prerequisites:
1. Anthropic API key (sign up at https://console.anthropic.com/)
2. Set ANTHROPIC_API_KEY environment variable

Features demonstrated:
- Basic Claude integration using OpenAI compatibility
- Extended thinking mode for complex reasoning
- Function calling with Claude
- Different Claude models (Opus, Sonnet, Haiku)
- Best practices for Anthropic limitations

Learn more:
- Anthropic OpenAI compatibility: https://docs.anthropic.com/en/api/openai-sdk
- Extended thinking: https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking
*/

var anthropicAPIKey = os.Getenv("ANTHROPIC_API_KEY")

func init() {
	if anthropicAPIKey == "" {
		fmt.Println("Please set ANTHROPIC_API_KEY environment variable")
		fmt.Println("Get your API key from: https://console.anthropic.com/")
		os.Exit(1)
	}

	// Disable OpenAI tracing since we're using Anthropic
	tracing.SetTracingDisabled(true)
}

// AnthropicProvider creates an OpenAI-compatible provider for Anthropic Claude
func createAnthropicProvider() *agents.OpenAIProvider {
	return agents.NewOpenAIProvider(agents.OpenAIProviderParams{
		BaseURL: param.NewOpt("https://api.anthropic.com/v1/"),
		APIKey:  param.NewOpt(anthropicAPIKey),
		// Anthropic doesn't support OpenAI Responses API yet
		UseResponses: param.NewOpt(false),
	})
}

// Research tool for demonstration
type ResearchArgs struct {
	Topic string `json:"topic" description:"The topic to research"`
	Depth string `json:"depth" description:"Research depth: 'basic', 'detailed', or 'comprehensive'"`
}

func conductResearch(ctx context.Context, args ResearchArgs) (string, error) {
	fmt.Printf("🔍 [Research Tool] Researching: %s (depth: %s)\n", args.Topic, args.Depth)

	// Simulate research based on depth
	var findings string
	switch args.Depth {
	case "basic":
		findings = fmt.Sprintf("Basic overview of %s: This is a fundamental concept with key applications in various fields.", args.Topic)
	case "detailed":
		findings = fmt.Sprintf("Detailed analysis of %s: This topic involves complex interactions between multiple factors, with significant implications for current practices and future developments.", args.Topic)
	case "comprehensive":
		findings = fmt.Sprintf("Comprehensive study of %s: An in-depth examination reveals multi-layered complexities, historical context, current state-of-the-art, emerging trends, and potential future directions with cross-disciplinary implications.", args.Topic)
	default:
		findings = fmt.Sprintf("Research findings for %s: General information and insights.", args.Topic)
	}

	return findings, nil
}

var researchTool = agents.NewFunctionTool("conduct_research", "Research a topic with specified depth", conductResearch)

func main() {
	fmt.Println("🤖 Anthropic Claude Provider Example")
	fmt.Println(strings.Repeat("=", 50))

	// Example 1: Basic Claude Usage
	fmt.Println("\n📝 Example 1: Basic Claude Integration")
	fmt.Println(strings.Repeat("-", 40))
	runBasicClaudeExample()

	// Example 2: Extended Thinking Mode
	fmt.Println("\n🧠 Example 2: Extended Thinking Mode")
	fmt.Println(strings.Repeat("-", 40))
	runExtendedThinkingExample()

	// Example 3: Function Calling with Claude
	fmt.Println("\n🛠️ Example 3: Function Calling")
	fmt.Println(strings.Repeat("-", 40))
	runFunctionCallingExample()

	// Example 4: Different Claude Models
	fmt.Println("\n🎭 Example 4: Different Claude Models")
	fmt.Println(strings.Repeat("-", 40))
	runDifferentModelsExample()

	fmt.Println("\n✅ All examples completed successfully!")
}

// Example 1: Basic Claude integration
func runBasicClaudeExample() {
	provider := createAnthropicProvider()

	agent := agents.New("Claude Assistant").
		WithInstructions("You are Claude, an AI assistant by Anthropic. Be helpful, harmless, and honest.")

	result, err := (agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("claude-3-5-sonnet-20241022")),
		},
	}).Run(context.Background(), agent, "Explain the concept of quantum computing in simple terms.")

	if err != nil {
		log.Printf("❌ Error in basic example: %v", err)
		return
	}

	fmt.Printf("🤖 Claude Response: %s\n", result.FinalOutput)
}

// Example 2: Extended thinking mode for complex reasoning
func runExtendedThinkingExample() {
	provider := createAnthropicProvider()

	agent := agents.New("Claude Researcher").
		WithInstructions("You are a thoughtful researcher. Think step by step through complex problems.").
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.7),
			MaxTokens:   param.NewOpt(int64(1500)),
			// Note: Extended thinking can be enabled via ExtraHeaders if supported
			ExtraHeaders: map[string]string{
				"anthropic-thinking": "enabled",
			},
		})

	result, err := (agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("claude-3-5-sonnet-20241022")),
		},
	}).Run(context.Background(), agent, "Analyze the potential impacts of artificial general intelligence on society, considering both positive and negative consequences.")

	if err != nil {
		log.Printf("❌ Error in thinking example: %v", err)
		return
	}

	fmt.Printf("🧠 Claude's Analysis: %s\n", result.FinalOutput)
}

// Example 3: Function calling with Claude
func runFunctionCallingExample() {
	provider := createAnthropicProvider()

	agent := agents.New("Research Assistant").
		WithInstructions("You are a research assistant. Use the research tool to gather information before answering questions.").
		WithTools(researchTool)

	result, err := (agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("claude-3-5-sonnet-20241022")),
		},
	}).Run(context.Background(), agent, "I need to understand machine learning. Can you research this topic comprehensively and then explain the key concepts?")

	if err != nil {
		log.Printf("❌ Error in function calling example: %v", err)
		return
	}

	fmt.Printf("🛠️ Research Result: %s\n", result.FinalOutput)
}

// Example 4: Different Claude models comparison
func runDifferentModelsExample() {
	provider := createAnthropicProvider()
	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	prompt := "Write a haiku about artificial intelligence."

	models := []string{
		"claude-3-5-sonnet-20241022", // Latest Sonnet (balanced)
		"claude-3-5-haiku-20241022",  // Haiku (fast, efficient)
		"claude-3-opus-20240229",     // Opus (most capable)
	}

	for _, model := range models {
		fmt.Printf("\n🎭 Testing model: %s\n", model)

		agent := agents.New("Poet").
			WithInstructions("You are a creative poet. Be concise and artistic.")

		result, err := runner.Run(context.Background(),
			agent.WithModel(model),
			prompt)

		if err != nil {
			log.Printf("❌ Error with model %s: %v", model, err)
			continue
		}

		fmt.Printf("📝 Result: %s\n", result.FinalOutput)
	}
}
