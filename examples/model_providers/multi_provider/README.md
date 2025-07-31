# Multi-Provider Example

This example demonstrates how to use multiple AI providers simultaneously with the OpenAI Agents Go SDK using prefix-based routing. This powerful pattern allows you to optimize for different use cases by routing tasks to the most suitable provider and model.

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐
│  Go Application │────│  MultiProvider   │
│  (Agents SDK)   │    │  (Prefix Router) │
└─────────────────┘    └─────────────────┘
                                │
                       ┌────────┼────────┐
                       │        │        │
              ┌─────────▼───┐ ┌──▼──┐ ┌───▼────────┐
              │ Anthropic   │ │ API │ │ OpenRouter │
              │ Claude      │ │     │ │ (Llama,    │
              │ (Reasoning) │ │     │ │ Gemma, etc)│
              └─────────────┘ └─────┘ └────────────┘
```

## 🚀 Quick Start

### Prerequisites

Set environment variables for the providers you want to use:

```bash
# Required for Anthropic Claude
export ANTHROPIC_API_KEY="sk-ant-your-key"

# Required for Mistral AI  
export MISTRAL_API_KEY="your-mistral-key"

# Required for OpenRouter (meta-provider)
export OPENROUTER_API_KEY="sk-or-your-key"

# Optional for OpenAI fallback
export OPENAI_API_KEY="sk-your-openai-key"
```

### Run the Example

```bash
cd examples/model_providers/multi_provider
go run main.go
```

## 📋 Features Demonstrated

### 1. Provider-Specific Task Routing

The example shows how different providers excel at different tasks:

```go
tasks := []Task{
    {
        Name:    "Reasoning Challenge",
        BestFor: "anthropic/claude-3-5-sonnet-20241022", // Claude for logic
    },
    {
        Name:    "Code Generation", 
        BestFor: "mistral/codestral-latest", // Mistral for code
    },
    {
        Name:    "Fast Response",
        BestFor: "openrouter/meta-llama/llama-3.1-8b-instruct", // Fast model
    },
}
```

### 2. Multi-Provider Setup

```go
func setupMultiProvider() *agents.MultiProvider {
    providerMap := agents.NewMultiProviderMap()
    
    // Add each provider with a prefix
    providerMap.AddProvider("anthropic", anthropicProvider)
    providerMap.AddProvider("mistral", mistralProvider) 
    providerMap.AddProvider("openrouter", openrouterProvider)
    
    return agents.NewMultiProvider(agents.NewMultiProviderParams{
        ProviderMap: providerMap,
    })
}
```

### 3. Prefix-Based Model Selection

```go
// Use Claude for reasoning
agent.WithModel("anthropic/claude-3-5-sonnet-20241022")

// Use Mistral for code
agent.WithModel("mistral/codestral-latest")

// Use Llama via OpenRouter
agent.WithModel("openrouter/meta-llama/llama-3.1-8b-instruct")
```

### 4. Provider Performance Comparison

The example includes a comparison feature that runs the same prompt across different providers to compare:
- Response quality
- Response time
- Approach differences

### 5. Tool Usage Across Providers

Demonstrates how function calling works consistently across different providers:

```go
agent := agents.New("Code Reviewer").
    WithTools(codeAnalysisTool).
    WithModel("mistral/mistral-large-latest") // Use Mistral for code analysis
```

## 🎯 Provider Strengths

### Anthropic Claude
- **Best for**: Complex reasoning, safety-critical tasks, analysis
- **Models**: `claude-3-5-sonnet-20241022`, `claude-3-opus-20240229`
- **Features**: Extended thinking, excellent reasoning, safety

### Mistral AI  
- **Best for**: Code generation, multilingual tasks, efficient inference
- **Models**: `mistral-large-latest`, `codestral-latest`, `mistral-small-latest`
- **Features**: Strong coding capabilities, multilingual support

### OpenRouter
- **Best for**: Access to many models, cost optimization, experimentation
- **Models**: `meta-llama/llama-3.1-8b-instruct`, `google/gemma-2-9b-it`, many others
- **Features**: Meta-provider access, competitive pricing

### OpenAI (Fallback)
- **Best for**: Baseline performance, reliable fallback
- **Models**: `gpt-4`, `gpt-3.5-turbo`
- **Features**: Proven reliability, comprehensive features

## 🔧 Configuration Options

### Task-Specific Optimization

```go
// For reasoning tasks
agent.WithModelSettings(modelsettings.ModelSettings{
    Temperature: param.NewOpt(0.7),
    ExtraBody: map[string]any{
        "thinking": map[string]any{"type": "enabled"}, // Claude thinking
    },
})

// For code tasks  
agent.WithModelSettings(modelsettings.ModelSettings{
    Temperature: param.NewOpt(0.2), // Lower temperature for precision
    MaxTokens:   param.NewOpt(int64(2000)),
})
```

### Fallback Strategy

```go
multiProvider := agents.NewMultiProvider(agents.NewMultiProviderParams{
    ProviderMap:  providerMap,
    OpenaiAPIKey: param.NewOpt(openaiKey), // OpenAI as fallback
})
```

## 🚀 Advanced Patterns

### 1. Cost Optimization

Route cheaper models for simple tasks:

```go
func selectModel(complexity string) string {
    switch complexity {
    case "simple":
        return "openrouter/meta-llama/llama-3.1-8b-instruct" // Cheap & fast
    case "medium":
        return "mistral/mistral-large-latest" // Balanced
    case "complex":
        return "anthropic/claude-3-opus-20240229" // Most capable
    }
}
```

### 2. Parallel Processing

Run the same task on multiple providers simultaneously:

```go
type ProviderResult struct {
    Provider string
    Result   string
    Duration time.Duration
    Error    error
}

func runParallel(prompt string) []ProviderResult {
    // Implementation to run across multiple providers concurrently
}
```

### 3. Dynamic Provider Selection

```go
func selectBestProvider(taskType string) string {
    providerMap := map[string]string{
        "reasoning": "anthropic/claude-3-5-sonnet-20241022",
        "coding":    "mistral/codestral-latest", 
        "creative":  "anthropic/claude-3-opus-20240229",
        "factual":   "openrouter/meta-llama/llama-3.1-8b-instruct",
    }
    return providerMap[taskType]
}
```

## 📊 Performance Guidelines

### When to Use Each Provider

| Task Type | Primary Choice | Alternative | Reason |
|-----------|---------------|-------------|---------|
| Complex reasoning | Anthropic Claude | OpenAI GPT-4 | Advanced reasoning capabilities |
| Code generation | Mistral Codestral | OpenAI GPT-4 | Specialized for code |
| Fast responses | OpenRouter Llama | Mistral Small | Speed optimization |
| Multilingual | Mistral Large | Anthropic Claude | Native multilingual training |
| Safety-critical | Anthropic Claude | OpenAI GPT-4 | Constitutional AI training |

### Cost Optimization

1. **Use cheaper models for simple tasks**: Llama 3.1 8B via OpenRouter
2. **Use specialized models for specific domains**: Codestral for code
3. **Use premium models only when needed**: Claude Opus for complex reasoning

## 🐛 Troubleshooting

### Common Issues

1. **Provider not available**
   ```
   Error: unknown prefix "provider"
   ```
   **Solution**: Check API key is set and provider is added to ProviderMap

2. **Model not found**
   ```
   Error: model "xyz" not found
   ```
   **Solution**: Verify model name format: `provider/model-name`

3. **Rate limiting**
   ```
   Error: 429 Too Many Requests
   ```
   **Solution**: Implement retry logic or spread load across providers

### Debug Tips

```go
// Log provider selection
fmt.Printf("Using provider: %s\n", modelName)

// Measure response times
start := time.Now()
result, err := runner.Run(context.Background(), agent, prompt)
fmt.Printf("Duration: %v\n", time.Since(start))
```

## 🌟 Benefits

1. **Optimization**: Use the best model for each task type
2. **Reliability**: Fallback options if a provider is unavailable  
3. **Cost Control**: Route to cheaper providers when appropriate
4. **Experimentation**: Easy to test different providers
5. **Scalability**: Distribute load across multiple providers

This multi-provider approach gives you the flexibility to build robust, optimized AI applications that leverage the strengths of different providers and models.