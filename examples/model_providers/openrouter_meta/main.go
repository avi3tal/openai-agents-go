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
OpenRouter Meta-Provider Example

OpenRouter is a unified API that provides access to 100+ language models from various providers
including OpenAI, Anthropic, Meta, Google, Mistral, and many others. This example demonstrates
how to use OpenRouter as a single gateway to access multiple model families.

Benefits of OpenRouter:
1. Single API key for 100+ models
2. Competitive pricing with automatic routing
3. Access to latest models from all major providers
4. Built-in load balancing and fallbacks
5. Detailed usage analytics

Prerequisites:
1. OpenRouter API key (sign up at https://openrouter.ai/)
2. Set OPENROUTER_API_KEY environment variable

Popular model categories available:
- OpenAI: GPT-4, GPT-3.5-turbo
- Anthropic: Claude-3.5-Sonnet, Claude-3-Opus
- Meta: Llama 3.1 (8B, 70B, 405B)
- Google: Gemini Pro, Gemma 2
- Mistral: Mistral Large, Codestral
- Specialized: Code Llama, Dolphin, Nous Research models
*/

var openrouterAPIKey = os.Getenv("OPENROUTER_API_KEY")

func init() {
	if openrouterAPIKey == "" {
		fmt.Println("Please set OPENROUTER_API_KEY environment variable")
		fmt.Println("Get your API key from: https://openrouter.ai/keys")
		os.Exit(1)
	}

	tracing.SetTracingDisabled(true)
}

// Model represents an available model with its characteristics
type Model struct {
	Name        string
	Provider    string
	Size        string
	Strengths   []string
	CostLevel   string // "low", "medium", "high"
	ContextSize string
}

// TaskBenchmark represents a task for benchmarking different models
type TaskBenchmark struct {
	Name         string
	Category     string
	Prompt       string
	ExpectedTime time.Duration
	Criteria     []string
}

// Translation tool for demonstration
type TranslateArgs struct {
	Text       string `json:"text" description:"Text to translate"`
	TargetLang string `json:"target_lang" description:"Target language (e.g., 'french', 'spanish', 'german')"`
}

func translateText(ctx context.Context, args TranslateArgs) (string, error) {
	fmt.Printf("🌍 [Translation Tool] Translating to %s: %s\n", args.TargetLang, args.Text[:min(50, len(args.Text))])

	// Simulate translation
	time.Sleep(300 * time.Millisecond)

	return fmt.Sprintf("Translated text to %s: [This would be the actual translation in a real implementation]", args.TargetLang), nil
}

var translationTool = agents.NewFunctionTool("translate_text", "Translate text to a specified language", translateText)

// createOpenRouterProvider creates an OpenRouter provider
func createOpenRouterProvider() *agents.OpenAIProvider {
	return agents.NewOpenAIProvider(agents.OpenAIProviderParams{
		BaseURL:      param.NewOpt("https://openrouter.ai/api/v1"),
		APIKey:       param.NewOpt(openrouterAPIKey),
		UseResponses: param.NewOpt(false), // Most providers don't support OpenAI Responses API
	})
}

func main() {
	fmt.Println("🚀 OpenRouter Meta-Provider Example")
	fmt.Println(strings.Repeat("=", 50))

	provider := createOpenRouterProvider()

	// Example 1: Model Showcase
	fmt.Println("\n🎭 Example 1: Model Showcase")
	fmt.Println(strings.Repeat("-", 40))
	showcaseModels(provider)

	// Example 2: Performance Comparison
	fmt.Println("\n🏁 Example 2: Performance Benchmarking")
	fmt.Println(strings.Repeat("-", 40))
	benchmarkModels(provider)

	// Example 3: Cost-Performance Analysis
	fmt.Println("\n💰 Example 3: Cost-Performance Analysis")
	fmt.Println(strings.Repeat("-", 40))
	analyzeCostPerformance(provider)

	// Example 4: Specialized Model Usage
	fmt.Println("\n🎯 Example 4: Specialized Model Usage")
	fmt.Println(strings.Repeat("-", 40))
	demonstrateSpecializedModels(provider)

	// Example 5: Tool Usage with Different Models
	fmt.Println("\n🛠️ Example 5: Tool Usage Across Models")
	fmt.Println(strings.Repeat("-", 40))
	demonstrateToolUsage(provider)

	fmt.Println("\n✅ OpenRouter examples completed!")
}

// Example 1: Showcase different model families available through OpenRouter
func showcaseModels(provider *agents.OpenAIProvider) {
	models := []Model{
		{
			Name:        "meta-llama/llama-3.1-8b-instruct",
			Provider:    "Meta",
			Size:        "8B",
			Strengths:   []string{"Fast", "General purpose", "Good reasoning"},
			CostLevel:   "low",
			ContextSize: "131K",
		},
		{
			Name:        "anthropic/claude-3.5-sonnet",
			Provider:    "Anthropic",
			Size:        "Unknown",
			Strengths:   []string{"Reasoning", "Safety", "Analysis"},
			CostLevel:   "high",
			ContextSize: "200K",
		},
		{
			Name:        "google/gemini-pro",
			Provider:    "Google",
			Size:        "Unknown",
			Strengths:   []string{"Multimodal", "Fast", "Factual"},
			CostLevel:   "medium",
			ContextSize: "128K",
		},
		{
			Name:        "mistralai/mistral-large",
			Provider:    "Mistral",
			Size:        "Unknown",
			Strengths:   []string{"Code", "Multilingual", "Efficient"},
			CostLevel:   "medium",
			ContextSize: "128K",
		},
	}

	prompt := "Explain the concept of recursion in programming with a simple example."

	agent := agents.New("Programming Tutor").
		WithInstructions("You are a helpful programming instructor. Provide clear, concise explanations with examples.")

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	for _, model := range models {
		fmt.Printf("\n🤖 Testing: %s (%s)\n", model.Name, model.Provider)
		fmt.Printf("📊 Strengths: %s | Cost: %s | Context: %s\n",
			strings.Join(model.Strengths, ", "), model.CostLevel, model.ContextSize)

		start := time.Now()
		result, err := runner.Run(context.Background(),
			agent.WithModel(model.Name),
			prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error with %s: %v", model.Name, err)
			continue
		}

		fmt.Printf("⏱️ Response time: %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📝 Response preview: %s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 150))
	}
}

// Example 2: Benchmark different models on various tasks
func benchmarkModels(provider *agents.OpenAIProvider) {
	benchmarks := []TaskBenchmark{
		{
			Name:     "Math Problem",
			Category: "reasoning",
			Prompt:   "If a train travels 120 km in 1.5 hours, what is its average speed in km/h?",
			Criteria: []string{"Accuracy", "Clear explanation"},
		},
		{
			Name:     "Code Writing",
			Category: "programming",
			Prompt:   "Write a Python function to find the factorial of a number using recursion.",
			Criteria: []string{"Correctness", "Code quality", "Documentation"},
		},
		{
			Name:     "Creative Writing",
			Category: "creativity",
			Prompt:   "Write a haiku about artificial intelligence.",
			Criteria: []string{"Creativity", "Adherence to format"},
		},
	}

	// Test with fast and capable models
	testModels := []string{
		"meta-llama/llama-3.1-8b-instruct", // Fast & cheap
		"anthropic/claude-3.5-sonnet",      // High quality
	}

	agent := agents.New("Benchmark Assistant").
		WithInstructions("Provide accurate, well-formatted responses.")

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	for _, benchmark := range benchmarks {
		fmt.Printf("\n📋 Benchmark: %s (%s)\n", benchmark.Name, benchmark.Category)
		fmt.Printf("✅ Criteria: %s\n", strings.Join(benchmark.Criteria, ", "))

		for _, model := range testModels {
			fmt.Printf("\n  🤖 %s:\n", model)

			start := time.Now()
			result, err := runner.Run(context.Background(),
				agent.WithModel(model),
				benchmark.Prompt)
			duration := time.Since(start)

			if err != nil {
				log.Printf("    ❌ Error: %v", err)
				continue
			}

			fmt.Printf("    ⏱️ Time: %v\n", duration.Round(time.Millisecond))
			fmt.Printf("    📝 Result: %s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 100))
		}
	}
}

// Example 3: Analyze cost vs performance tradeoffs
func analyzeCostPerformance(provider *agents.OpenAIProvider) {
	// Simulate a cost-sensitive scenario
	prompt := "Summarize the key benefits of renewable energy in 2-3 sentences."

	// Models with different cost/performance profiles
	costAnalysis := []struct {
		Model       string
		CostTier    string
		Description string
	}{
		{
			Model:       "meta-llama/llama-3.1-8b-instruct",
			CostTier:    "💚 Low Cost",
			Description: "Fast, efficient, good for simple tasks",
		},
		{
			Model:       "mistralai/mistral-large",
			CostTier:    "🟡 Medium Cost",
			Description: "Balanced performance and cost",
		},
		{
			Model:       "anthropic/claude-3.5-sonnet",
			CostTier:    "🔴 High Cost",
			Description: "Premium quality, best for complex tasks",
		},
	}

	agent := agents.New("Efficiency Analyst").
		WithInstructions("Provide concise, accurate information.")

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	fmt.Println("Cost-Performance Analysis for: \"" + prompt + "\"")

	for _, analysis := range costAnalysis {
		fmt.Printf("\n%s - %s\n", analysis.CostTier, analysis.Model)
		fmt.Printf("📋 %s\n", analysis.Description)

		start := time.Now()
		result, err := runner.Run(context.Background(),
			agent.WithModel(analysis.Model),
			prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("⏱️ Response time: %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📝 Quality assessment: %s\n", assessResponseQuality(fmt.Sprintf("%v", result.FinalOutput)))
		fmt.Printf("💬 Response: %s\n", fmt.Sprintf("%v", result.FinalOutput))
	}
}

// Example 4: Demonstrate specialized models for specific tasks
func demonstrateSpecializedModels(provider *agents.OpenAIProvider) {
	specializedTasks := []struct {
		Task      string
		Model     string
		Prompt    string
		Reasoning string
	}{
		{
			Task:      "Code Generation",
			Model:     "meta-llama/codellama-34b-instruct",
			Prompt:    "Create a Go struct for a user with validation methods.",
			Reasoning: "Code Llama is specialized for programming tasks",
		},
		{
			Task:      "Mathematical Reasoning",
			Model:     "anthropic/claude-3.5-sonnet",
			Prompt:    "Solve this step by step: If f(x) = 2x² + 3x - 1, find f'(x) and f'(2).",
			Reasoning: "Claude excels at step-by-step mathematical reasoning",
		},
		{
			Task:      "Fast General Query",
			Model:     "meta-llama/llama-3.1-8b-instruct",
			Prompt:    "What are the main differences between REST and GraphQL APIs?",
			Reasoning: "Llama 3.1 8B is fast and efficient for straightforward questions",
		},
	}

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	for _, task := range specializedTasks {
		fmt.Printf("\n🎯 Task: %s\n", task.Task)
		fmt.Printf("🤖 Specialized Model: %s\n", task.Model)
		fmt.Printf("💡 Why: %s\n", task.Reasoning)

		agent := agents.New("Specialist").
			WithInstructions("You are an expert in this domain. Provide detailed, accurate responses.")

		start := time.Now()
		result, err := runner.Run(context.Background(),
			agent.WithModel(task.Model),
			task.Prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("⏱️ Duration: %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📝 Result: %s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 200))
	}
}

// Example 5: Tool usage across different models
func demonstrateToolUsage(provider *agents.OpenAIProvider) {
	agent := agents.New("Multilingual Assistant").
		WithInstructions("You are a helpful multilingual assistant. Use the translation tool when users need translations.").
		WithTools(translationTool).
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.3),
		})

	// Test tool usage with different models
	models := []string{
		"meta-llama/llama-3.1-8b-instruct",
		"mistralai/mistral-large", // Good for multilingual tasks
	}

	prompt := "I need to translate 'Hello, how are you today?' to Spanish and French for my international presentation."

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	for _, model := range models {
		fmt.Printf("\n🛠️ Testing tool usage with: %s\n", model)

		result, err := runner.Run(context.Background(),
			agent.WithModel(model),
			prompt)

		if err != nil {
			log.Printf("❌ Error with %s: %v", model, err)
			continue
		}

		fmt.Printf("🌍 Translation result: %s\n", fmt.Sprintf("%v", result.FinalOutput))
	}
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func assessResponseQuality(response string) string {
	length := len(response)
	if length < 50 {
		return "Brief response"
	} else if length < 200 {
		return "Adequate detail"
	} else {
		return "Comprehensive response"
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
