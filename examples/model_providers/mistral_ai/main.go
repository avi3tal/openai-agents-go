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
	"regexp"
	"strings"
	"time"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/modelsettings"
	"github.com/nlpodyssey/openai-agents-go/tracing"
	"github.com/openai/openai-go/packages/param"
)

/*
Mistral AI Provider Example

This example demonstrates how to use Mistral AI's models through their OpenAI-compatible API.
Mistral is known for:

1. Excellent code generation capabilities (Codestral)
2. Strong multilingual support (trained on diverse language data)
3. Efficient inference with good performance-to-cost ratio
4. Function calling and structured outputs
5. Both open-source and API-available models

Available Mistral Models:
- mistral-large-latest: Most capable, best for complex reasoning
- mistral-medium-latest: Balanced performance and speed
- mistral-small-latest: Fast and efficient for simpler tasks
- codestral-latest: Specialized for code generation and understanding
- mistral-embed: For embeddings and similarity tasks

Prerequisites:
1. Mistral AI API key (sign up at https://console.mistral.ai/)
2. Set MISTRAL_API_KEY environment variable

Learn more:
- Mistral AI: https://mistral.ai/
- API Documentation: https://docs.mistral.ai/
- Model comparison: https://docs.mistral.ai/getting-started/models/
*/

var mistralAPIKey = os.Getenv("MISTRAL_API_KEY")

func init() {
	if mistralAPIKey == "" {
		fmt.Println("Please set MISTRAL_API_KEY environment variable")
		fmt.Println("Get your API key from: https://console.mistral.ai/")
		os.Exit(1)
	}

	tracing.SetTracingDisabled(true)
}

// CodeReview tool for demonstration
type CodeReviewArgs struct {
	Code     string `json:"code" description:"Code to review"`
	Language string `json:"language" description:"Programming language"`
	Focus    string `json:"focus" description:"Review focus: 'security', 'performance', 'style', or 'all'"`
}

func reviewCode(ctx context.Context, args CodeReviewArgs) (string, error) {
	fmt.Printf("🔍 [Code Review] Analyzing %s code (focus: %s)\n", args.Language, args.Focus)

	// Simulate code review
	time.Sleep(400 * time.Millisecond)

	review := fmt.Sprintf(`
Code Review Results (%s focus):
✅ Strengths: Code structure is clear and follows %s conventions
⚠️  Suggestions: Consider adding error handling and documentation
🔧 Optimizations: Variable naming could be more descriptive
🛡️  Security: No obvious security issues detected
📊 Overall Score: 8/10`, args.Focus, args.Language)

	return review, nil
}

var codeReviewTool = agents.NewFunctionTool("review_code", "Perform code review with specific focus", reviewCode)

// Translation tool demonstrating multilingual capabilities
type MultilingualTranslateArgs struct {
	Text            string   `json:"text" description:"Text to translate"`
	TargetLanguages []string `json:"target_languages" description:"List of target languages"`
	Style           string   `json:"style" description:"Translation style: 'formal', 'casual', 'technical'"`
}

func multilingualTranslate(ctx context.Context, args MultilingualTranslateArgs) (string, error) {
	fmt.Printf("🌍 [Translator] Translating to %v (style: %s)\n", args.TargetLanguages, args.Style)

	time.Sleep(300 * time.Millisecond)

	result := fmt.Sprintf("Multilingual translation results (%s style):\n", args.Style)
	for _, lang := range args.TargetLanguages {
		result += fmt.Sprintf("- %s: [Translation would appear here]\n", lang)
	}

	return result, nil
}

var translationTool = agents.NewFunctionTool("multilingual_translate", "Translate text to multiple languages", multilingualTranslate)

// createMistralProvider creates a Mistral AI provider
func createMistralProvider() *agents.OpenAIProvider {
	return agents.NewOpenAIProvider(agents.OpenAIProviderParams{
		BaseURL:      param.NewOpt("https://api.mistral.ai/v1"),
		APIKey:       param.NewOpt(mistralAPIKey),
		UseResponses: param.NewOpt(false), // Mistral uses Chat Completions API
	})
}

func main() {
	fmt.Println("🚀 Mistral AI Provider Example")
	fmt.Println(strings.Repeat("=", 50))

	provider := createMistralProvider()

	// Example 1: Code Generation with Codestral
	fmt.Println("\n💻 Example 1: Code Generation with Codestral")
	fmt.Println(strings.Repeat("-", 40))
	demonstrateCodeGeneration(provider)

	// Example 2: Multilingual Capabilities
	fmt.Println("\n🌍 Example 2: Multilingual Support")
	fmt.Println(strings.Repeat("-", 40))
	demonstrateMultilingualCapabilities(provider)

	// Example 3: Model Comparison
	fmt.Println("\n⚖️ Example 3: Mistral Model Comparison")
	fmt.Println(strings.Repeat("-", 40))
	compareMistralModels(provider)

	// Example 4: Function Calling
	fmt.Println("\n🛠️ Example 4: Function Calling & Tools")
	fmt.Println(strings.Repeat("-", 40))
	demonstrateFunctionCalling(provider)

	// Example 5: Structured Output
	fmt.Println("\n📋 Example 5: Structured Output Generation")
	fmt.Println(strings.Repeat("-", 40))
	demonstrateStructuredOutput(provider)

	fmt.Println("\n✅ Mistral AI examples completed!")
}

// Example 1: Code generation using Codestral
func demonstrateCodeGeneration(provider *agents.OpenAIProvider) {
	codeRequests := []struct {
		Task        string
		Language    string
		Description string
		Prompt      string
	}{
		{
			Task:        "API Handler",
			Language:    "Go",
			Description: "REST API endpoint with validation",
			Prompt:      "Create a Go HTTP handler for a user registration endpoint with JSON validation, error handling, and proper responses.",
		},
		{
			Task:        "Algorithm Implementation",
			Language:    "Python",
			Description: "Efficient sorting algorithm",
			Prompt:      "Implement a merge sort algorithm in Python with detailed comments and time complexity analysis.",
		},
		{
			Task:        "React Component",
			Language:    "TypeScript",
			Description: "Interactive UI component",
			Prompt:      "Create a TypeScript React component for a searchable dropdown with proper type definitions and error handling.",
		},
	}

	agent := agents.New("Senior Developer").
		WithInstructions("You are an expert software developer. Write clean, well-documented, production-ready code with error handling and best practices.").
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.2), // Lower temperature for more consistent code
			MaxTokens:   param.NewOpt(int64(2000)),
		})

	runner := agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("codestral-latest")),
		},
	}

	for _, req := range codeRequests {
		fmt.Printf("\n📝 Task: %s (%s)\n", req.Task, req.Language)
		fmt.Printf("📋 Description: %s\n", req.Description)

		start := time.Now()
		result, err := runner.Run(context.Background(), agent, req.Prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("⏱️ Generation time: %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📊 Code quality: %s\n", assessCodeQuality(fmt.Sprintf("%v", result.FinalOutput)))
		fmt.Printf("💻 Generated code preview:\n%s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 300))
	}
}

// Example 2: Multilingual capabilities
func demonstrateMultilingualCapabilities(provider *agents.OpenAIProvider) {
	multilingualTasks := []struct {
		Name     string
		Prompt   string
		Expected string
	}{
		{
			Name:     "French Technical Documentation",
			Prompt:   "Expliquez en français comment fonctionne l'algorithme de tri rapide (quicksort).",
			Expected: "French explanation",
		},
		{
			Name:     "Spanish Code Comments",
			Prompt:   "Escribe comentarios en español para esta función de Python: def fibonacci(n): return n if n <= 1 else fibonacci(n-1) + fibonacci(n-2)",
			Expected: "Spanish comments",
		},
		{
			Name:     "German Problem Solving",
			Prompt:   "Löse dieses mathematische Problem auf Deutsch: Wenn ein Zug 240 km in 3 Stunden fährt, wie hoch ist seine Durchschnittsgeschwindigkeit?",
			Expected: "German solution",
		},
		{
			Name:     "Code in Multiple Languages",
			Prompt:   "Write a 'Hello World' program in Python, then explain it in French, Spanish, and German.",
			Expected: "Multilingual explanation",
		},
	}

	agent := agents.New("Multilingual Expert").
		WithInstructions("You are a multilingual technical expert. Respond in the requested language with native fluency.")

	runner := agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("mistral-large-latest")),
		},
	}

	for _, task := range multilingualTasks {
		fmt.Printf("\n🌍 Task: %s\n", task.Name)

		start := time.Now()
		result, err := runner.Run(context.Background(), agent, task.Prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("⏱️ Response time: %v\n", duration.Round(time.Millisecond))
		fmt.Printf("📝 Response: %s\n", truncateString(fmt.Sprintf("%v", result.FinalOutput), 200))
	}
}

// Example 3: Compare different Mistral models
func compareMistralModels(provider *agents.OpenAIProvider) {
	models := []struct {
		Name        string
		Description string
		BestFor     string
	}{
		{
			Name:        "mistral-large-latest",
			Description: "Most capable Mistral model",
			BestFor:     "Complex reasoning, analysis, coding",
		},
		{
			Name:        "mistral-medium-latest",
			Description: "Balanced performance and efficiency",
			BestFor:     "General purpose, balanced tasks",
		},
		{
			Name:        "mistral-small-latest",
			Description: "Fast and cost-efficient",
			BestFor:     "Simple tasks, high-volume usage",
		},
		{
			Name:        "codestral-latest",
			Description: "Specialized for code",
			BestFor:     "Code generation, understanding, review",
		},
	}

	testPrompt := "Explain the difference between synchronous and asynchronous programming with a code example."

	agent := agents.New("Technical Educator").
		WithInstructions("Provide clear, educational explanations with practical examples.")

	runner := agents.Runner{Config: agents.RunConfig{ModelProvider: provider}}

	for _, model := range models {
		fmt.Printf("\n🤖 Model: %s\n", model.Name)
		fmt.Printf("📋 Description: %s\n", model.Description)
		fmt.Printf("🎯 Best for: %s\n", model.BestFor)

		start := time.Now()
		result, err := runner.Run(context.Background(),
			agent.WithModel(model.Name),
			testPrompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("⏱️ Response time: %v\n", duration.Round(time.Millisecond))
		finalOutput := fmt.Sprintf("%v", result.FinalOutput)
		fmt.Printf("📏 Response length: %d chars\n", len(finalOutput))
		fmt.Printf("📝 Quality: %s\n", assessResponseQuality(finalOutput))
		fmt.Printf("🔍 Preview: %s\n", truncateString(finalOutput, 150))
	}
}

// Example 4: Function calling with tools
func demonstrateFunctionCalling(provider *agents.OpenAIProvider) {
	agent := agents.New("Code Reviewer & Translator").
		WithInstructions("You are a senior developer and linguist. Use tools to help with code review and translations.").
		WithTools(codeReviewTool, translationTool).
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.3),
		})

	tasks := []struct {
		Name   string
		Prompt string
	}{
		{
			Name: "Code Review Task",
			Prompt: `Please review this Go code for security issues:
			
func processUserInput(input string) string {
    return fmt.Sprintf("Hello %s", input)
}`,
		},
		{
			Name:   "Translation Task",
			Prompt: "Translate 'Welcome to our application' to French, Spanish, and German in a formal style.",
		},
	}

	runner := agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("mistral-large-latest")),
		},
	}

	for _, task := range tasks {
		fmt.Printf("\n🛠️ %s\n", task.Name)

		result, err := runner.Run(context.Background(), agent, task.Prompt)
		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("📋 Result: %s\n", fmt.Sprintf("%v", result.FinalOutput))
	}
}

// Example 5: Structured output generation
func demonstrateStructuredOutput(provider *agents.OpenAIProvider) {
	agent := agents.New("Data Analyst").
		WithInstructions("You are a data analyst. Provide structured, well-formatted responses with clear sections and bullet points.").
		WithModelSettings(modelsettings.ModelSettings{
			Temperature: param.NewOpt(0.1), // Very low for consistent structure
			MaxTokens:   param.NewOpt(int64(1500)),
		})

	structuredTasks := []struct {
		Name   string
		Prompt string
		Format string
	}{
		{
			Name:   "API Documentation",
			Format: "Structured API docs",
			Prompt: "Generate API documentation for a user management endpoint. Include: endpoint details, parameters, response format, error codes, and examples.",
		},
		{
			Name:   "Code Architecture Analysis",
			Format: "Technical analysis",
			Prompt: "Analyze the pros and cons of microservices vs monolithic architecture. Structure your response with clear sections for: Overview, Advantages, Disadvantages, Use Cases, and Recommendations.",
		},
		{
			Name:   "Project Plan",
			Format: "Structured plan",
			Prompt: "Create a project plan for developing a mobile app. Include: project overview, phases, timeline, resources needed, and risk assessment.",
		},
	}

	runner := agents.Runner{
		Config: agents.RunConfig{
			ModelProvider: provider,
			Model:         param.NewOpt(agents.NewAgentModelName("mistral-large-latest")),
		},
	}

	for _, task := range structuredTasks {
		fmt.Printf("\n📋 %s (%s)\n", task.Name, task.Format)

		start := time.Now()
		result, err := runner.Run(context.Background(), agent, task.Prompt)
		duration := time.Since(start)

		if err != nil {
			log.Printf("❌ Error: %v", err)
			continue
		}

		fmt.Printf("⏱️ Generation time: %v\n", duration.Round(time.Millisecond))
		finalOutput := fmt.Sprintf("%v", result.FinalOutput)
		fmt.Printf("📊 Structure quality: %s\n", assessStructureQuality(finalOutput))
		fmt.Printf("📝 Output:\n%s\n", finalOutput)
	}
}

// Helper functions
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

func assessCodeQuality(code string) string {
	var score int

	if strings.Contains(code, "func ") || strings.Contains(code, "def ") {
		score++
	}
	if strings.Contains(code, "//") || strings.Contains(code, "#") {
		score++
	}
	if strings.Contains(code, "error") || strings.Contains(code, "Error") {
		score++
	}
	if strings.Contains(code, "return") {
		score++
	}

	switch score {
	case 4:
		return "Excellent (complete with comments & error handling)"
	case 3:
		return "Good (well-structured)"
	case 2:
		return "Fair (basic structure)"
	default:
		return "Basic"
	}
}

func assessResponseQuality(response string) string {
	length := len(response)

	if length < 100 {
		return "Brief"
	} else if length < 500 {
		return "Adequate"
	} else if length < 1000 {
		return "Comprehensive"
	} else {
		return "Detailed"
	}
}

func assessStructureQuality(text string) string {
	var score int

	// Check for structured elements
	if strings.Contains(text, "##") || strings.Contains(text, "###") {
		score++
	}
	if strings.Contains(text, "- ") || strings.Contains(text, "* ") {
		score++
	}
	if strings.Contains(text, "1.") || strings.Contains(text, "2.") {
		score++
	}
	if regexp.MustCompile(`\*\*.*\*\*`).MatchString(text) {
		score++
	}

	switch score {
	case 4:
		return "Excellent (well-structured with headers, lists, and formatting)"
	case 3:
		return "Good (clear structure)"
	case 2:
		return "Fair (some structure)"
	default:
		return "Basic (minimal structure)"
	}
}
