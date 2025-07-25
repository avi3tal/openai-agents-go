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
	"os"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/tracing"
	"github.com/openai/openai-go/packages/param"
)

var (
	BaseURL   = os.Getenv("EXAMPLE_BASE_URL")
	APIKey    = os.Getenv("EXAMPLE_API_KEY")
	ModelName = os.Getenv("EXAMPLE_MODEL_NAME")
)

func init() {
	if BaseURL == "" || APIKey == "" || ModelName == "" {
		fmt.Println("Please set EXAMPLE_BASE_URL, EXAMPLE_API_KEY, EXAMPLE_MODEL_NAME via env var or code.")
		os.Exit(1)
	}
}

/*
This example uses a custom provider for all requests by default. We do three things:
1. Create a custom client.
2. Set it as the default OpenAI client.
3. Set the default API as Chat Completions, as most LLM providers don't yet support Responses API.

Note that in this example, we disable tracing under the assumption that you don't have an API key
from platform.openai.com. If you do have one, you can either set the `OPENAI_API_KEY` env var
or call tracing.SetTracingExportAPIKey to set a tracing specific key.
*/

var Client = agents.NewOpenaiClient(param.NewOpt(BaseURL), param.NewOpt(APIKey))

func init() {
	agents.SetDefaultOpenaiClient(Client, false)
	agents.SetDefaultOpenaiAPI(agents.OpenaiAPITypeChatCompletions)
	tracing.SetTracingDisabled(true)
}

type GetWeatherArgs struct {
	City string `json:"city"`
}

func GetWeather(_ context.Context, args GetWeatherArgs) (string, error) {
	fmt.Printf("[debug] getting weather for %s\n", args.City)
	return fmt.Sprintf("The weather in %s is sunny.", args.City), nil
}

var GetWeatherTool = agents.NewFunctionTool("get_weather", "", GetWeather)

func main() {
	agent := agents.New("Assistant").
		WithInstructions("You only respond in haikus.").
		WithModel(ModelName).
		WithTools(GetWeatherTool)

	result, err := agents.Run(context.Background(), agent, "What's the weather in Tokyo?")
	if err != nil {
		panic(err)
	}
	fmt.Println(result.FinalOutput)
}
