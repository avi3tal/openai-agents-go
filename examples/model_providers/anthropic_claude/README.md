# Anthropic Claude Provider Example

This example demonstrates how to integrate Anthropic's Claude models with the OpenAI Agents Go SDK using Anthropic's OpenAI SDK compatibility layer.

## 🏗️ Architecture

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│  Go Application │────│  Anthropic API  │────│   Claude Models │
│  (Agents SDK)   │    │  (OpenAI compat)│    │ (Opus/Sonnet/   │
└─────────────────┘    └─────────────────┘    │  Haiku)         │
                                               └─────────────────┘
```

## 🚀 Quick Start

### Prerequisites

1. **Anthropic API Key** - Sign up at [Anthropic Console](https://console.anthropic.com/)
2. **Go 1.21+** for running the example

### Step 1: Set Your API Key

```bash
export ANTHROPIC_API_KEY="sk-ant-your-anthropic-api-key-here"
```

### Step 2: Run the Example

```bash
cd examples/model_providers/anthropic_claude
go run main.go
```

## 📋 Example Features

The example demonstrates **four different Claude integration patterns**:

### 1. Basic Claude Integration
```go
provider := agents.NewOpenAIProvider(agents.OpenAIProviderParams{
    BaseURL:      param.NewOpt("https://api.anthropic.com/v1/"),
    APIKey:       param.NewOpt(anthropicAPIKey),
    UseResponses: param.NewOpt(false), // Anthropic limitation
})

agent := agents.New("Claude Assistant").
    WithInstructions("You are Claude, helpful and honest.")

result, err := (agents.Runner{
    Config: agents.RunConfig{
        ModelProvider: provider,
        Model: param.NewOpt(agents.NewAgentModelName("claude-3-5-sonnet-20241022")),
    },
}).Run(context.Background(), agent, "Your question here")
```

### 2. Extended Thinking Mode
```go
agent := agents.New("Claude Researcher").
    WithModelSettings(modelsettings.ModelSettings{
        ExtraBody: map[string]any{
            "thinking": map[string]any{
                "type":          "enabled",
                "budget_tokens": 2000,
            },
        },
    })
```

### 3. Function Calling
```go
agent := agents.New("Research Assistant").
    WithTools(researchTool). // Claude supports function calling
    WithInstructions("Use tools to gather information.")
```

### 4. Multiple Claude Models
- **Claude-3.5-Sonnet**: Balanced performance and capability
- **Claude-3.5-Haiku**: Fast and efficient for simpler tasks  
- **Claude-3-Opus**: Most capable for complex reasoning

## 🔧 Configuration

### Available Claude Models

| Model | Best For | Context | Strengths |
|-------|----------|---------|-----------|
| `claude-3-5-sonnet-20241022` | General use | 200K | Balanced speed/capability |
| `claude-3-5-haiku-20241022` | Fast responses | 200K | Speed, efficiency |
| `claude-3-opus-20240229` | Complex tasks | 200K | Advanced reasoning |
| `claude-opus-4-20250514` | Latest & best | 200K | Top performance |

### Extended Thinking Configuration

Enable Claude's step-by-step reasoning for complex tasks:

```go
modelSettings := modelsettings.ModelSettings{
    ExtraBody: map[string]any{
        "thinking": map[string]any{
            "type":          "enabled",
            "budget_tokens": 2000, // Adjust based on complexity
        },
    },
    Temperature: param.NewOpt(0.7),
    MaxTokens:   param.NewOpt(int64(1500)),
}
```

## ⚠️ Important Limitations

Anthropic's OpenAI compatibility has some limitations:

### API Behavior Differences
- **No `strict` parameter**: Function calling JSON isn't guaranteed to follow schema exactly
- **Audio input ignored**: Audio inputs are stripped from requests
- **No prompt caching**: Use native Anthropic SDK for prompt caching features
- **System message hoisting**: All system/developer messages are concatenated and moved to the beginning

### Recommended Settings
```go
// Always use for Anthropic
UseResponses: param.NewOpt(false)

// Disable OpenAI tracing
tracing.SetTracingDisabled(true)
```

## 🎯 Advanced Features

### Custom Headers
```go
modelSettings := modelsettings.ModelSettings{
    ExtraHeaders: map[string]string{
        "anthropic-version": "2023-06-01",
        "x-custom-header":   "value",
    },
}
```

### Error Handling
```go
result, err := runner.Run(context.Background(), agent, prompt)
if err != nil {
    // Anthropic errors follow OpenAI format but with different messages
    log.Printf("Claude error: %v", err)
    return
}
```

## 🐛 Troubleshooting

### Common Issues

1. **Invalid API Key**
   ```
   Error: 401 Unauthorized
   ```
   **Solution**: Verify your `ANTHROPIC_API_KEY` is correct and active.

2. **Model Not Found**
   ```
   Error: model "xyz" not found
   ```
   **Solution**: Use exact model names like `claude-3-5-sonnet-20241022`.

3. **Rate Limits**
   ```
   Error: 429 Too Many Requests
   ```
   **Solution**: Implement exponential backoff or upgrade your Anthropic plan.

### Debug Mode

Enable verbose logging:
```go
// Add debug logging
log.SetLevel(log.DebugLevel)

// Monitor requests
fmt.Printf("Making request to model: %s\n", modelName)
```

## 🌟 Benefits of Claude Integration

1. **Advanced Reasoning**: Claude excels at complex analysis and step-by-step thinking
2. **Safety**: Built-in safety measures and constitutional AI training
3. **Large Context**: 200K token context window for long documents
4. **Multimodal**: Support for text and image inputs (on supported models)
5. **Function Calling**: Reliable tool use and structured outputs

## 🔗 Learn More

- [Anthropic OpenAI SDK Compatibility](https://docs.anthropic.com/en/api/openai-sdk)
- [Extended Thinking Documentation](https://docs.anthropic.com/en/docs/build-with-claude/extended-thinking)
- [Claude Model Comparison](https://docs.anthropic.com/en/docs/about-claude/models)
- [Function Calling Guide](https://docs.anthropic.com/en/docs/build-with-claude/tool-use)

## 📊 Performance Comparison

When to choose Claude models:

- **Complex reasoning tasks**: Use Claude-3-Opus or latest Claude-4
- **Balanced performance**: Use Claude-3.5-Sonnet
- **Fast responses**: Use Claude-3.5-Haiku  
- **Safety-critical applications**: All Claude models excel here
- **Long document analysis**: Leverage the 200K context window