# Workflow Runner Manifests

This directory will contain declarative JSON manifests that mirror the Go examples under
`examples/`. Each manifest describes a `workflowrunner.WorkflowRequest` and can be executed with:

```
go run ./workflowrunner/examples/run_manifest path/to/manifest.json [stdout]
```

Pass `stdout` as the optional second argument (or set `WORKFLOWRUNNER_STDOUT=true`) to override the
manifest’s callback configuration and print events locally. You can also set the manifest path via
the `WORKFLOW_MANIFEST` environment variable.

During the migration effort, keep an index of the ported examples here and add new manifests next to
this README. Use `-` as the manifest path to read from stdin.

## Ported manifests

- `basic_hello_world.json` – mirrors `examples/basic/hello_world`
- `basic_tools.json` – mirrors `examples/basic/tools` (requires `get_weather` function tool)
- `basic_dynamic_system_prompt.json` – mirrors `examples/basic/dynamic_system_prompt`
- `basic_input_list.json` – mirrors `examples/basic/input_list`
- `basic_reasoning_usage.json` – mirrors `examples/basic/reasoning_usage`
- `basic_remote_image.json` – mirrors `examples/basic/remote_image`
- `agent_patterns_agents_as_tools.json` – mirrors `examples/agent_patterns/agents_as_tools`
- `agent_patterns_input_guardrails.json` – mirrors `examples/agent_patterns/input_guardrails`
- `agent_patterns_output_guardrails.json` – mirrors `examples/agent_patterns/output_guardrails` (uses the `sensitive_data_check` guardrail registered in `run_manifest`)
- `tools_web_search.json` – mirrors `examples/tools/web_search`
- `tools_code_interpreter.json` – mirrors `examples/tools/code_interpreter`
- `approvals_hosted_mcp.json` – interactive hosted MCP approval demo
