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

package agents

import (
	"fmt"

	"github.com/nlpodyssey/openai-agents-go/openaitypes"
	"github.com/openai/openai-go/responses"
)

// RunItem is an item generated by an agent.
type RunItem interface {
	isRunItem()
	ToInputItem() TResponseInputItem
}

// MessageOutputItem represents a message from the LLM.
type MessageOutputItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw response output message.
	RawItem responses.ResponseOutputMessage

	// Always `message_output_item`.
	Type string
}

func (MessageOutputItem) isRunItem() {}

func (item MessageOutputItem) ToInputItem() TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseOutputMessage(item.RawItem)
}

// HandoffCallItem represents a tool call for a handoff from one agent to another.
type HandoffCallItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw response function tool call that represents the handoff.
	RawItem responses.ResponseFunctionToolCall

	// Always `handoff_call_item`.
	Type string
}

func (HandoffCallItem) isRunItem() {}

func (item HandoffCallItem) ToInputItem() TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseFunctionToolCall(item.RawItem)
}

// HandoffOutputItem represents the output of a handoff.
type HandoffOutputItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw input item that represents the handoff taking place.
	RawItem TResponseInputItem

	// The agent that made the handoff.
	SourceAgent *Agent

	// The agent that is being handed off to.
	TargetAgent *Agent

	// Always `handoff_output_item`.
	Type string
}

func (HandoffOutputItem) isRunItem() {}

func (item HandoffOutputItem) ToInputItem() TResponseInputItem {
	return item.RawItem
}

// ToolCallItem represents a tool call e.g. a function call or computer action call.
type ToolCallItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw tool call item.
	RawItem ToolCallItemType

	// Always `tool_call_item`.
	Type string
}

func (ToolCallItem) isRunItem() {}

func (item ToolCallItem) ToInputItem() TResponseInputItem {
	return TResponseInputItemFromToolCallItemType(item.RawItem)
}

// ToolCallItemType is a type that represents a tool call item.
type ToolCallItemType interface {
	isToolCallItemType()
}

type ResponseFunctionToolCall responses.ResponseFunctionToolCall

func (ResponseFunctionToolCall) isToolCallItemType() {}

type ResponseComputerToolCall responses.ResponseComputerToolCall

func (ResponseComputerToolCall) isToolCallItemType() {}

type ResponseOutputItemLocalShellCall responses.ResponseOutputItemLocalShellCall

func (ResponseOutputItemLocalShellCall) isToolCallItemType() {}

type ResponseFileSearchToolCall responses.ResponseFileSearchToolCall

func (ResponseFileSearchToolCall) isToolCallItemType() {}

type ResponseFunctionWebSearch responses.ResponseFunctionWebSearch

func (ResponseFunctionWebSearch) isToolCallItemType() {}

type ResponseCodeInterpreterToolCall responses.ResponseCodeInterpreterToolCall

func (ResponseCodeInterpreterToolCall) isToolCallItemType() {}

type ResponseOutputItemImageGenerationCall responses.ResponseOutputItemImageGenerationCall

func (ResponseOutputItemImageGenerationCall) isToolCallItemType() {}

type ResponseOutputItemMcpCall responses.ResponseOutputItemMcpCall

func (ResponseOutputItemMcpCall) isToolCallItemType() {}

func TResponseInputItemFromToolCallItemType(input ToolCallItemType) TResponseInputItem {
	switch v := input.(type) {
	case ResponseFunctionToolCall:
		return TResponseInputItemFromResponseFunctionToolCall(v)
	case ResponseComputerToolCall:
		return TResponseInputItemFromResponseComputerToolCall(v)
	case ResponseOutputItemLocalShellCall:
		return TResponseInputItemFromResponseOutputItemLocalShellCall(v)
	default:
		// This would be an unrecoverable implementation bug, so a panic is appropriate.
		panic(fmt.Errorf("unexpected ToolCallItemType type %T", v))
	}
}

func TResponseInputItemFromResponseFunctionToolCall(input ResponseFunctionToolCall) TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseFunctionToolCall(responses.ResponseFunctionToolCall(input))
}

func TResponseInputItemFromResponseComputerToolCall(input ResponseComputerToolCall) TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseComputerToolCall(responses.ResponseComputerToolCall(input))
}

func TResponseInputItemFromResponseOutputItemLocalShellCall(input ResponseOutputItemLocalShellCall) TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseOutputItemLocalShellCall(responses.ResponseOutputItemLocalShellCall(input))
}

// ToolCallOutputItem represents the output of a tool call.
type ToolCallOutputItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw item from the model.
	RawItem ToolCallOutputRawItem

	// The output of the tool call. This is whatever the tool call returned; the `raw_item`
	// contains a string representation of the output.
	Output any

	// Always `tool_call_output_item`.
	Type string
}

type ToolCallOutputRawItem interface {
	isToolCallOutputRawItem()
}

type ResponseInputItemFunctionCallOutputParam responses.ResponseInputItemFunctionCallOutputParam

func (ResponseInputItemFunctionCallOutputParam) isToolCallOutputRawItem() {}

type ResponseInputItemComputerCallOutputParam responses.ResponseInputItemComputerCallOutputParam

func (ResponseInputItemComputerCallOutputParam) isToolCallOutputRawItem() {}

type ResponseInputItemLocalShellCallOutputParam responses.ResponseInputItemLocalShellCallOutputParam

func (ResponseInputItemLocalShellCallOutputParam) isToolCallOutputRawItem() {}

func (ToolCallOutputItem) isRunItem() {}

func (item ToolCallOutputItem) ToInputItem() TResponseInputItem {
	switch rawItem := item.RawItem.(type) {
	case ResponseInputItemFunctionCallOutputParam:
		return openaitypes.ResponseInputItemUnionParamFromResponseInputItemFunctionCallOutputParam(
			responses.ResponseInputItemFunctionCallOutputParam(rawItem))
	case ResponseInputItemComputerCallOutputParam:
		return openaitypes.ResponseInputItemUnionParamFromResponseInputItemComputerCallOutputParam(
			responses.ResponseInputItemComputerCallOutputParam(rawItem))
	case ResponseInputItemLocalShellCallOutputParam:
		return openaitypes.ResponseInputItemUnionParamFromResponseInputItemLocalShellCallOutputParam(
			responses.ResponseInputItemLocalShellCallOutputParam(rawItem))
	default:
		// This would be an unrecoverable implementation bug, so a panic is appropriate.
		panic(fmt.Errorf("unexpected ToolCallOutputRawItem type %T", rawItem))
	}
}

// ReasoningItem represents a reasoning item.
type ReasoningItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw reasoning item.
	RawItem responses.ResponseReasoningItem

	// Always `reasoning_item`.
	Type string
}

func (ReasoningItem) isRunItem() {}

func (item ReasoningItem) ToInputItem() TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseReasoningItem(item.RawItem)
}

// MCPListToolsItem represents a call to an MCP server to list tools.
type MCPListToolsItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw MCP list tools call.
	RawItem responses.ResponseOutputItemMcpListTools

	// Always `mcp_list_tools_item`.
	Type string
}

func (MCPListToolsItem) isRunItem() {}

func (item MCPListToolsItem) ToInputItem() TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseOutputItemMcpListTools(item.RawItem)
}

// MCPApprovalRequestItem represents a request for MCP approval.
type MCPApprovalRequestItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw MCP approval request.
	RawItem responses.ResponseOutputItemMcpApprovalRequest

	// Always `mcp_approval_request_item`.
	Type string
}

func (MCPApprovalRequestItem) isRunItem() {}

func (item MCPApprovalRequestItem) ToInputItem() TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseOutputItemMcpApprovalRequest(item.RawItem)
}

// MCPApprovalResponseItem represents a response to an MCP approval request.
type MCPApprovalResponseItem struct {
	// The agent whose run caused this item to be generated.
	Agent *Agent

	// The raw MCP approval response.
	RawItem responses.ResponseInputItemMcpApprovalResponseParam

	// Always `mcp_approval_response_item`.
	Type string
}

func (MCPApprovalResponseItem) isRunItem() {}

func (item MCPApprovalResponseItem) ToInputItem() TResponseInputItem {
	return openaitypes.ResponseInputItemUnionParamFromResponseInputItemMcpApprovalResponseParam(item.RawItem)
}
