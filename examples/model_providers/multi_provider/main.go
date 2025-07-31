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
	"time"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/modelsettings"
	"github.com/nlpodyssey/openai-agents-go/tracing"
	"github.com/openai/openai-go/packages/param"
)

/*
Multi-Provider Example

This example demonstrates how to use multiple AI providers simultaneously using the SDK's
MultiProvider system with prefix-based routing. This allows you to:

1. Route different tasks to the best-suited provider/model
2. Implement fallback strategies
3. Compare outputs from different providers
4. Optimize costs by using cheaper providers when appropriate

Providers demonstrated:
- Anthropic Claude (reasoning, safety)
- Mistral AI (code generation, multilingual)
- OpenRouter (meta-provider with many models)
- OpenAI (baseline comparison)

Prerequisites:
Set environment variables for the providers you want to use:
- ANTHROPIC_API_KEY
- MISTRAL_API_KEY
- OPENROUTER_API_KEY
- OPENAI_API_KEY (optional)
*/

var (
	anthropicKey  = os.Getenv("ANTHROPIC_API_KEY")
	mistralKey    = os.Getenv("MISTRAL_API_KEY")
	openrouterKey = os.Getenv("OPENROUTER_API_KEY")
	openaiKey     = os.Getenv("OPENAI_API_KEY")
)

func init() {
	tracing.SetTracingDisabled(true)
}

// Task represents a task to be completed by an AI model
type Task struct {
	Name        string
	Description string
	Prompt      string
	BestFor     string // Which provider/model is best suited
}

// CodeAnalysis tool for demonstration
type CodeAnalysisArgs struct {
	Code     string `json:"code" description:"The code to analyze"`
	Language string `json:"language" description:"Programming language (e.g., 'python', 'go', 'javascript')"`
}

func analyzeCode(ctx context.Context, args CodeAnalysisArgs) (string, error) {
	fmt.Printf("🔍 [Code Analysis] Analyzing %s code...\n", args.Language)

	// Simulate code analysis
	time.Sleep(500 * time.Millisecond)

	analysis := fmt.Sprintf(`
Code Analysis for %s:
- Structure: Well-organized with clear function separation
- Complexity: Medium level complexity
- Potential Issues: Minor optimization opportunities identified
- Best Practices: Generally follows %s conventions
- Security: No obvious security vulnerabilities detected
`, args.Language, args.Language)

	return analysis, nil
}

var codeAnalysisTool = agents.NewFunctionTool("analyze_code", "Analyze code for structure, issues, and best practices", analyzeCode)

// setupMultiProvider configures providers with prefix routing
func setupMultiProvider() *agents.MultiProvider {
	providerMap := agents.NewMultiProviderMap()

	// Add Anthropic Claude - excellent for reasoning and safety
	if anthropicKey != "" {
		fmt.Println("✅ Adding Anthropic Claude provider")
		providerMap.AddProvider("anthropic", agents.NewOpenAIProvider(agents.OpenAIProviderParams{
			BaseURL:      param.NewOpt("https://api.anthropic.com/v1/"),
			APIKey:       param.NewOpt(anthropicKey),
			UseResponses: param.NewOpt(false),
		}))
	} else {
		fmt.Println("⚠️ ANTHROPIC_API_KEY not set, skipping Anthropic provider")
	}

	// Add Mistral AI - excellent for code generation and multilingual tasks
	if mistralKey != "" {
		fmt.Println("✅ Adding Mistral AI provider")
		providerMap.AddProvider("mistral", agents.NewOpenAIProvider(agents.OpenAIProviderParams{
			BaseURL:      param.NewOpt("https://api.mistral.ai/v1/"),
			APIKey:       param.NewOpt(mistralKey),
			UseResponses: param.NewOpt(false),
		}))
	} else {
		fmt.Println("⚠️ MISTRAL_API_KEY not set, skipping Mistral provider")
	}

	// Add OpenRouter - meta-provider with access to many models
	if openrouterKey != "" {
		fmt.Println("✅ Adding OpenRouter meta-provider")
		providerMap.AddProvider("openrouter", agents.NewOpenAIProvider(agents.OpenAIProviderParams{
			BaseURL:      param.NewOpt("https://openrouter.ai/api/v1/"),
			APIKey:       param.NewOpt(openrouterKey),
			UseResponses: param.NewOpt(false),
		}))
	} else {
		fmt.Println("⚠️ OPENROUTER_API_KEY not set, skipping OpenRouter provider")
	}

	// OpenAI as fallback (will be used if no prefix matches or as default)
	return agents.NewMultiProvider(agents.NewMultiProviderParams{
		ProviderMap:   providerMap,
		OpenaiAPIKey:  param.NewOpt(openaiKey),
		OpenaiBaseURL: param.NewOpt(""), // Use default OpenAI endpoint
	})
}

func main() {
	fmt.Println("🌍 Multi-Provider AI Example")
	fmt.Println(strings.Repeat("=", 50))

	multiProvider := setupMultiProvider()

	// Define various tasks that suit different providers
	tasks := []Task{
		{
			Name:        "Reasoning Challenge",
			Description: "Complex logical reasoning task",
			Prompt:      "A farmer has 17 sheep. All but 9 die. How many are left? Explain your reasoning step by step.",
			BestFor:     "anthropic/claude-3-5-sonnet-20241022", // Claude excels at reasoning
		},
		{
			Name:        "Code Generation",
			Description: "Generate optimized code",
			Prompt:      "Write a Go function that implements a binary search algorithm with proper error handling and documentation.",
			BestFor:     "mistral/codestral-latest", // Mistral's code-focused model
		},
		{
			Name:        "Creative Writing",
			Description: "Creative content generation",
			Prompt:      "Write a short story about a robot learning to paint, exploring themes of creativity and consciousness.",
			BestFor:     "anthropic/claude-3-opus-20240229", // Claude Opus for creativity
		},
		{
			Name:        "Fast Response",
			Description: "Quick factual response",
			Prompt:      "What are the three largest cities in Japan by population?",
			BestFor:     "openrouter/meta-llama/llama-3.1-8b-instruct", // Fast model via OpenRouter
		},
		{
			Name:        "Multilingual Task",
			Description: "Multilingual understanding",
			Prompt:      "Translate this text to French, German, and Spanish: 'The future of artificial intelligence is bright.'",
			BestFor:     "mistral/mistral-large-latest", // Mistral's multilingual strength
		},
	}

	// Run tasks with their optimal providers
	fmt.Println("\n🎯 Running Tasks with Optimal Providers")
	fmt.Println(strings.Repeat("-", 50))

	for i, task := range tasks {
		fmt.Printf("\n📋 Task %d: %s\n", i+1, task.Name)
		fmt.Printf("🎯 Best model: %s\n", task.BestFor)
		fmt.Printf("📝 Description: %s\n", task.Description)

		runTask(multiProvider, task)
	}

	// Demonstrate provider comparison
	fmt.Println("\n🆚 Provider Comparison")
	fmt.Println(strings.Repeat("-", 50))
	compareProviders(multiProvider)

	// Demonstrate agent with tool usage
	fmt.Println("\n🛠️ Multi-Provider Tool Usage")
	fmt.Println(strings.Repeat("-", 50))
	demonstrateToolUsage(multiProvider)

	fmt.Println("\n✅ Multi-provider example completed!")
}

func runTask(provider *agents.MultiProvider, task Task) {
	agent := agents.New("Specialist Assistant").
		WithInstructions("You are an expert assistant. Provide high-quality, accurate responses.")

	runner := agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName(task.BestFor)),
		},
	}

	start := time.Now()
	result, err := runner.Run(context.Background(), agent, task.Prompt)
	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ Error running task '%s': %v", task.Name, err)
		return
	}

	fmt.Printf("⏱️ Duration: %v\n", duration.Round(time.Millisecond))
	fmt.Printf("🤖 Response: %s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 200))
}

func compareProviders(provider *agents.MultiProvider) {
	prompt := "Explain quantum computing in exactly 3 sentences."

	models := []string{
		"anthropic/claude-3-5-sonnet-20241022",
		"mistral/mistral-large-latest",
		"openrouter/meta-llama/llama-3.1-8b-instruct",
	}

	agent := agents.New("Explainer").
		WithInstructions("You are a science communicator. Be precise and clear.")

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	for _, model := range models {
		fmt.Printf("\n🔍 Testing: %s\n", model)

		start := time.Now()
		result, err := runner.Run(context.Background(),
			agent.WithModel(model),
			prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error with %s: %v", model, err)
			continue
		}

		fmt.Printf("⏱️ Duration: %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📝 Response: %s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 150))
	}
}

func demonstrateToolUsage(provider *agents.MultiProvider) {
	// Use Mistral for code analysis (good at code understanding)
	agent := agents.New("Code Reviewer").
		WithInstructions("You are an expert code reviewer. Use the code analysis tool to examine code and provide detailed feedback.").
		WithTools(codeAnalysisTool).
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.3), // Lower temperature for technical tasks
		})

	codeToAnalyze := `
func fibonacci(n int) int {
    if n <= 1 {
        return n
    }
    return fibonacci(n-1) + fibonacci(n-2)
}
`

	result, err := (agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("mistral/mistral-large-latest")),
		},
	}).Run(context.Background(), agent,
		fmt.Sprintf("Please analyze this Go code and suggest improvements: %s", codeToAnalyze))

	if err != nil {
		log.Printf("❌ Error in tool usage example: %v", err)
		return
	}

	fmt.Printf("🛠️ Code Review Result: %s\n", result.FinalOutput)
}

// Helper function to truncate long strings for display
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
