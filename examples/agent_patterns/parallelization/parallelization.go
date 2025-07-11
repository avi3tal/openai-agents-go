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
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/nlpodyssey/openai-agents-go/tracing"
)

/*
This example shows the parallelization pattern. We run the agent three times
in parallel, and pick the best result.
*/

const Model = "gpt-4.1-nano"

var (
	SpanishAgent = agents.New("spanish_agent").
			WithInstructions("You translate the user's message to Spanish").
			WithModel(Model)
	TranslationPicker = agents.New("translation_picker").
				WithInstructions("You pick the best Spanish translation from the given options.").
				WithModel(Model)
)

func main() {
	fmt.Print("Hi! Enter a message, and we'll translate it to Spanish.\n\n")
	_ = os.Stdout.Sync()
	line, _, err := bufio.NewReader(os.Stdin).ReadLine()
	if err != nil {
		panic(err)
	}
	msg := string(line)

	// Ensure the entire workflow is a single trace
	err = tracing.RunTrace(
		context.Background(),
		tracing.TraceParams{WorkflowName: "Parallel translation"},
		func(ctx context.Context, _ tracing.Trace) error {
			const N = 3
			var runResults [N]*agents.RunResult
			var runErrors [N]error

			var wg sync.WaitGroup
			wg.Add(N)

			for i := range N {
				go func() {
					defer wg.Done()
					runResults[i], runErrors[i] = agents.Run(ctx, SpanishAgent, msg)
				}()
			}

			wg.Wait()
			if err = errors.Join(runErrors[:]...); err != nil {
				return err
			}

			var outputs [N]string
			for i, runResult := range runResults {
				outputs[i] = agents.ItemHelpers().TextMessageOutputs(runResult.NewItems)
			}

			translations := strings.Join(outputs[:], "\n\n")
			fmt.Printf("\n\nTranslations:\n\n%s\n", translations)

			input := fmt.Sprintf("Input: %s\n\nTranslations:\n%s", msg, translations)
			bestTranslation, err := agents.Run(ctx, TranslationPicker, input)
			if err != nil {
				return err
			}

			fmt.Println("\n\n-----")
			fmt.Printf("Best translation: %s\n", bestTranslation.FinalOutput)
			return nil
		},
	)
	if err != nil {
		panic(err)
	}
}
