# OpenRouter Meta-Provider Example

This example demonstrates how to use OpenRouter as a unified gateway to access 100+ language models from various providers including OpenAI, Anthropic, Meta, Google, Mistral, and many others through a single API.

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Go Application │────│   OpenRouter    │────│   100+ Models   │
│  (Agents SDK)   │    │   (Unified API) │    │ (Multiple       │
└─────────────────┘    └─────────────────┘    │  Providers)     │
                                               └─────────────────┘
                                                        │
                        ┌───────────────────────────────┼───────────────────────────────┐
                        │                               │                               │
                ┌───────▼────────┐              ┌───────▼────────┐              ┌───────▼────────┐
                │    OpenAI      │              │   Anthropic    │              │     Meta       │
                │   GPT-4, 3.5   │              │ Claude 3.5, 3  │              │ Llama 3.1/3.2  │
                └────────────────┘              └────────────────┘              └────────────────┘
                        │                               │                               │
                ┌───────▼────────┐              ┌───────▼────────┐              ┌───────▼────────┐
                │     Google     │              │    Mistral     │              │  Specialized   │
                │ Gemini, Gemma  │              │  Large, Code   │              │ Code, Research │
                └────────────────┘              └────────────────┘              └────────────────┘
```

## 🚀 Quick Start

### Prerequisites

1. **OpenRouter API Key** - Sign up at [OpenRouter](https://openrouter.ai/)
2. **Go 1.21+** for running the example

### Step 1: Get Your API Key

```bash
# Sign up at https://openrouter.ai/
# Get your API key from https://openrouter.ai/keys
export OPENROUTER_API_KEY="sk-or-your-openrouter-api-key"
```

### Step 2: Run the Example

```bash
cd examples/model_providers/openrouter_meta
go run main.go
```

## 📋 Example Features

The example demonstrates **five key OpenRouter capabilities**:

### 1. Model Showcase
Compare responses from different model families:

```go
models := []Model{
    {Name: "meta-llama/llama-3.1-8b-instruct", Provider: "Meta"},
    {Name: "anthropic/claude-3.5-sonnet", Provider: "Anthropic"},
    {Name: "google/gemini-pro", Provider: "Google"},
    {Name: "mistralai/mistral-large", Provider: "Mistral"},
}
```

### 2. Performance Benchmarking
Test different models on standardized tasks:
- Mathematical reasoning
- Code generation  
- Creative writing
- Response time analysis

### 3. Cost-Performance Analysis
Compare models across different cost tiers:

```go
costAnalysis := []struct {
    Model       string
    CostTier    string
    Description string
}{
    {"meta-llama/llama-3.1-8b-instruct", "💚 Low Cost", "Fast, efficient"},
    {"mistralai/mistral-large", "🟡 Medium Cost", "Balanced performance"},
    {"anthropic/claude-3.5-sonnet", "🔴 High Cost", "Premium quality"},
}
```

### 4. Specialized Model Usage
Route tasks to models optimized for specific domains:
- **Code Llama**: Programming tasks
- **Claude**: Mathematical reasoning
- **Llama 3.1**: General queries

### 5. Tool Usage Across Models
Demonstrate function calling consistency across different providers.

## 🎯 Available Model Categories

### **OpenAI Models**
- `openai/gpt-4` - Most capable general model
- `openai/gpt-3.5-turbo` - Fast and efficient
- `openai/gpt-4-vision-preview` - Multimodal capabilities

### **Anthropic Claude Models**  
- `anthropic/claude-3.5-sonnet` - Latest balanced model
- `anthropic/claude-3-opus` - Most capable reasoning
- `anthropic/claude-3-haiku` - Fastest responses

### **Meta Llama Models**
- `meta-llama/llama-3.1-405b-instruct` - Largest open model
- `meta-llama/llama-3.1-70b-instruct` - High performance
- `meta-llama/llama-3.1-8b-instruct` - Fast and efficient
- `meta-llama/codellama-34b-instruct` - Code specialization

### **Google Models**
- `google/gemini-pro` - Multimodal capabilities
- `google/gemma-2-9b-it` - Efficient performance
- `google/gemma-2-27b-it` - Higher capability

### **Mistral Models**
- `mistralai/mistral-large` - Most capable Mistral model
- `mistralai/mistral-medium` - Balanced performance
- `mistralai/codestral-latest` - Code generation

### **Specialized Models**
- `microsoft/wizardlm-2-8x22b` - Advanced reasoning
- `nousresearch/nous-hermes-2-mixtral-8x7b-dpo` - Fine-tuned performance
- `togethercomputer/alpaca-7b` - Research-focused

## 🔧 Configuration

### Basic Setup

```go
provider := agents.NewOpenAIProvider(agents.OpenAIProviderParams{
    BaseURL:      param.NewOpt("https://openrouter.ai/api/v1"),
    APIKey:       param.NewOpt(openrouterAPIKey),
    UseResponses: param.NewOpt(false), // Most models don't support Responses API
})
```

### Model Selection Strategy

```go
func selectOptimalModel(taskType string, budget string) string {
    switch taskType {
    case "reasoning":
        if budget == "high" {
            return "anthropic/claude-3.5-sonnet"
        }
        return "meta-llama/llama-3.1-70b-instruct"
    case "coding":
        return "meta-llama/codellama-34b-instruct"
    case "creative":
        return "anthropic/claude-3-opus"
    case "fast":
        return "meta-llama/llama-3.1-8b-instruct"
    default:
        return "openai/gpt-3.5-turbo"
    }
}
```

### Custom Headers for OpenRouter

```go
modelSettings := modelsettings.ModelSettings{
    ExtraHeaders: map[string]string{
        "HTTP-Referer": "https://your-app.com",     // Optional: for app identification
        "X-Title":      "Your App Name",            // Optional: for analytics
    },
}
```

## 💰 Cost Optimization

### Understanding OpenRouter Pricing

| Cost Tier | Example Models | Best For |
|-----------|---------------|----------|
| **Free Tier** | Limited daily usage | Testing and evaluation |
| **Low Cost** | Llama 3.1 8B, Gemma 2 9B | High-volume simple tasks |
| **Medium Cost** | Mistral Large, GPT-3.5 | Balanced performance needs |
| **High Cost** | Claude 3.5 Sonnet, GPT-4 | Complex, critical tasks |

### Cost-Saving Strategies

```go
// 1. Use cheaper models for simple tasks
func getCheapModel() string {
    return "meta-llama/llama-3.1-8b-instruct"
}

// 2. Implement model fallbacks
func getModelWithFallback(preferred string) string {
    models := []string{
        preferred,
        "meta-llama/llama-3.1-8b-instruct", // Cheaper fallback
        "openai/gpt-3.5-turbo",             // Reliable fallback
    }
    // Try models in order based on availability/cost
    return models[0]
}

// 3. Cache responses for repeated queries
// (Implementation depends on your caching strategy)
```

## 🚀 Advanced Patterns

### 1. Dynamic Model Selection

```go
type TaskAnalyzer struct {
    complexity int
    domain     string
    budget     float64
}

func (ta TaskAnalyzer) SelectModel() string {
    if ta.domain == "code" {
        return "meta-llama/codellama-34b-instruct"
    }
    if ta.complexity > 8 && ta.budget > 0.01 {
        return "anthropic/claude-3.5-sonnet"
    }
    return "meta-llama/llama-3.1-8b-instruct"
}
```

### 2. A/B Testing Models

```go
func runABTest(prompt string) (map[string]string, error) {
    models := []string{
        "anthropic/claude-3.5-sonnet",
        "openai/gpt-4",
        "meta-llama/llama-3.1-70b-instruct",
    }
    
    results := make(map[string]string)
    for _, model := range models {
        result, err := runWithModel(model, prompt)
        if err != nil {
            continue
        }
        results[model] = result
    }
    return results, nil
}
```

### 3. Load Distribution

```go
func distributeLoad(requests []string) map[string][]string {
    distribution := map[string][]string{
        "fast_models":    {},  // For simple queries
        "balanced_models": {}, // For medium complexity
        "premium_models":  {}, // For complex tasks
    }
    
    for _, req := range requests {
        complexity := analyzeComplexity(req)
        switch {
        case complexity < 3:
            distribution["fast_models"] = append(distribution["fast_models"], req)
        case complexity < 7:
            distribution["balanced_models"] = append(distribution["balanced_models"], req)
        default:
            distribution["premium_models"] = append(distribution["premium_models"], req)
        }
    }
    return distribution
}
```

## 📊 Performance Benchmarks

### Response Time Comparison (Approximate)

| Model | Simple Query | Complex Reasoning | Code Generation |
|-------|-------------|------------------|-----------------|
| Llama 3.1 8B | ~0.5s | ~2s | ~1.5s |
| Mistral Large | ~1s | ~3s | ~2s |
| Claude 3.5 Sonnet | ~1.5s | ~4s | ~3s |
| GPT-4 | ~2s | ~5s | ~3.5s |

### Quality Assessment Matrix

| Task Type | Best Model | Alternative | Budget Option |
|-----------|------------|-------------|---------------|
| Reasoning | Claude 3.5 Sonnet | GPT-4 | Llama 3.1 70B |
| Code | Code Llama 34B | Claude 3.5 | Llama 3.1 8B |
| Creative | Claude 3 Opus | GPT-4 | Mistral Large |
| Fast/General | Llama 3.1 8B | GPT-3.5 | Gemma 2 9B |

## 🐛 Troubleshooting

### Common Issues

1. **Invalid Model Name**
   ```
   Error: model "xyz" not found
   ```
   **Solution**: Check [OpenRouter models page](https://openrouter.ai/models) for exact names

2. **Rate Limiting**
   ```
   Error: 429 Too Many Requests
   ```
   **Solution**: Implement exponential backoff or upgrade your plan

3. **Insufficient Credits**
   ```
   Error: 402 Payment Required
   ```
   **Solution**: Add credits to your OpenRouter account

### Debug Mode

```go
// Log model selection
fmt.Printf("Selected model: %s for task: %s\n", modelName, taskType)

// Monitor costs
fmt.Printf("Estimated cost: $%.6f\n", estimatedCost)

// Track performance
start := time.Now()
result, err := runner.Run(context.Background(), agent, prompt)
fmt.Printf("Response time: %v\n", time.Since(start))
```

## 🌟 Benefits of OpenRouter

1. **Unified Access**: Single API for 100+ models
2. **Cost Efficiency**: Competitive pricing with transparent costs
3. **Reliability**: Built-in fallbacks and load balancing
4. **Flexibility**: Easy switching between providers and models
5. **Analytics**: Detailed usage tracking and cost analysis
6. **Innovation**: Access to latest models as they're released

## 🔗 Learn More

- [OpenRouter Official Site](https://openrouter.ai/)
- [Available Models](https://openrouter.ai/models)
- [API Documentation](https://openrouter.ai/docs)
- [Pricing Information](https://openrouter.ai/pricing)
- [Model Comparison Tool](https://openrouter.ai/playground)

OpenRouter is an excellent choice when you need flexibility, cost control, and access to the latest AI models without managing multiple API keys and integrations.