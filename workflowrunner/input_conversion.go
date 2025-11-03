package workflowrunner

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared/constant"
)

func buildInputItems(inputs []WorkflowInput) ([]agents.TResponseInputItem, error) {
	if len(inputs) == 0 {
		return nil, nil
	}
	items := make([]agents.TResponseInputItem, 0, len(inputs))
	for idx, input := range inputs {
		item, err := workflowInputToResponseItem(input)
		if err != nil {
			return nil, fmt.Errorf("inputs[%d]: %w", idx, err)
		}
		items = append(items, item)
	}
	return items, nil
}

func workflowInputToResponseItem(input WorkflowInput) (responses.ResponseInputItemUnionParam, error) {
	switch strings.ToLower(strings.TrimSpace(input.Type)) {
	case "message":
		msg, err := buildMessageInput(input)
		if err != nil {
			return responses.ResponseInputItemUnionParam{}, err
		}
		return responses.ResponseInputItemUnionParam{OfMessage: msg}, nil
	case "text":
		converted := input
		converted.Type = "message"
		if strings.TrimSpace(converted.Role) == "" {
			converted.Role = "user"
		}
		if converted.Content == nil && strings.TrimSpace(converted.URI) != "" {
			converted.Content = strings.TrimSpace(converted.URI)
			converted.URI = ""
		}
		if converted.Content == nil {
			return responses.ResponseInputItemUnionParam{}, fmt.Errorf("text input requires content or uri")
		}
		return workflowInputToResponseItem(converted)
	case "image":
		if strings.TrimSpace(input.URI) == "" && input.Content == nil {
			return responses.ResponseInputItemUnionParam{}, fmt.Errorf("image input requires uri or content")
		}
		message := WorkflowInput{
			Type: "message",
			Role: strings.TrimSpace(defaultString(input.Role, "user")),
		}
		content := []any{map[string]any{
			"type":      "input_image",
			"image_url": strings.TrimSpace(input.URI),
		}}
		if input.Content != nil {
			switch v := input.Content.(type) {
			case []any:
				content = v
			case map[string]any:
				content = []any{v}
			}
		}
		message.Content = content
		return workflowInputToResponseItem(message)
	default:
		return responses.ResponseInputItemUnionParam{}, fmt.Errorf("type %q not supported yet", input.Type)
	}
}

func buildMessageInput(input WorkflowInput) (*responses.EasyInputMessageParam, error) {
	role, err := normalizeMessageRole(input.Role)
	if err != nil {
		return nil, err
	}
	message := &responses.EasyInputMessageParam{
		Role: role,
		Type: responses.EasyInputMessageTypeMessage,
	}

	if input.Content == nil && strings.TrimSpace(input.URI) != "" {
		contentList, err := buildMessageContentList([]any{map[string]any{
			"type":      "input_image",
			"image_url": strings.TrimSpace(input.URI),
		}})
		if err != nil {
			return nil, err
		}
		message.Content = responses.EasyInputMessageContentUnionParam{
			OfInputItemContentList: contentList,
		}
		return message, nil
	}

	switch value := input.Content.(type) {
	case nil:
		return nil, fmt.Errorf("message input requires content")
	case string:
		message.Content = responses.EasyInputMessageContentUnionParam{
			OfString: param.NewOpt(value),
		}
	case []any:
		contentList, err := buildMessageContentList(value)
		if err != nil {
			return nil, err
		}
		message.Content = responses.EasyInputMessageContentUnionParam{
			OfInputItemContentList: contentList,
		}
	case map[string]any:
		if parts, ok := value["parts"]; ok {
			rawParts, ok := parts.([]any)
			if !ok {
				return nil, fmt.Errorf("message content parts must be an array")
			}
			contentList, err := buildMessageContentList(rawParts)
			if err != nil {
				return nil, err
			}
			message.Content = responses.EasyInputMessageContentUnionParam{
				OfInputItemContentList: contentList,
			}
			break
		}
		if text, ok := getString(value, "text"); ok && text != "" {
			message.Content = responses.EasyInputMessageContentUnionParam{
				OfString: param.NewOpt(text),
			}
			break
		}
		// Fallback: allow direct marshaling into the union structure.
		raw, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("marshal message content: %w", err)
		}
		var union responses.EasyInputMessageContentUnionParam
		if err := json.Unmarshal(raw, &union); err != nil {
			return nil, fmt.Errorf("decode message content: %w", err)
		}
		if !messageContentUnionHasValue(union) {
			return nil, fmt.Errorf("decode message content: missing expected fields")
		}
		message.Content = union
	default:
		return nil, fmt.Errorf("message content type %T not supported", value)
	}

	return message, nil
}

func buildMessageContentList(parts []any) (responses.ResponseInputMessageContentListParam, error) {
	if len(parts) == 0 {
		return nil, fmt.Errorf("message content list cannot be empty")
	}
	result := make(responses.ResponseInputMessageContentListParam, 0, len(parts))
	for idx, part := range parts {
		content, err := buildContentUnion(part)
		if err != nil {
			return nil, fmt.Errorf("content[%d]: %w", idx, err)
		}
		result = append(result, content)
	}
	return result, nil
}

func buildContentUnion(value any) (responses.ResponseInputContentUnionParam, error) {
	switch v := value.(type) {
	case string:
		return responses.ResponseInputContentUnionParam{
			OfInputText: &responses.ResponseInputTextParam{
				Type: constant.ValueOf[constant.InputText](),
				Text: v,
			},
		}, nil
	case map[string]any:
		kindValue, _ := getString(v, "type")
		kind := strings.ToLower(strings.TrimSpace(defaultString(kindValue, "input_text")))
		switch kind {
		case "input_text", "text":
			text, ok := getString(v, "text")
			if !ok || strings.TrimSpace(text) == "" {
				return responses.ResponseInputContentUnionParam{}, fmt.Errorf("text content requires text field")
			}
			return responses.ResponseInputContentUnionParam{
				OfInputText: &responses.ResponseInputTextParam{
					Type: constant.ValueOf[constant.InputText](),
					Text: text,
				},
			}, nil
		case "input_image", "image":
			imageParam := responses.ResponseInputImageParam{
				Type:   constant.ValueOf[constant.InputImage](),
				Detail: responses.ResponseInputImageDetailAuto,
			}
			if detail, ok := getString(v, "detail"); ok && detail != "" {
				switch strings.ToLower(detail) {
				case "low":
					imageParam.Detail = responses.ResponseInputImageDetailLow
				case "high":
					imageParam.Detail = responses.ResponseInputImageDetailHigh
				default:
					imageParam.Detail = responses.ResponseInputImageDetailAuto
				}
			}
			if url, ok := getString(v, "image_url"); ok && strings.TrimSpace(url) != "" {
				imageParam.ImageURL = param.NewOpt(strings.TrimSpace(url))
			}
			if fileID, ok := getString(v, "file_id"); ok && strings.TrimSpace(fileID) != "" {
				imageParam.FileID = param.NewOpt(strings.TrimSpace(fileID))
			}
			if !imageParam.ImageURL.Valid() && !imageParam.FileID.Valid() {
				return responses.ResponseInputContentUnionParam{}, fmt.Errorf("image content requires image_url or file_id")
			}
			return responses.ResponseInputContentUnionParam{
				OfInputImage: &imageParam,
			}, nil
		case "input_file", "file":
			fileParam := responses.ResponseInputFileParam{
				Type: constant.ValueOf[constant.InputFile](),
			}
			if url, ok := getString(v, "file_url"); ok && strings.TrimSpace(url) != "" {
				fileParam.FileURL = param.NewOpt(strings.TrimSpace(url))
			}
			if data, ok := getString(v, "file_data"); ok && strings.TrimSpace(data) != "" {
				fileParam.FileData = param.NewOpt(strings.TrimSpace(data))
			}
			if id, ok := getString(v, "file_id"); ok && strings.TrimSpace(id) != "" {
				fileParam.FileID = param.NewOpt(strings.TrimSpace(id))
			}
			if !fileParam.FileURL.Valid() && !fileParam.FileData.Valid() && !fileParam.FileID.Valid() {
				return responses.ResponseInputContentUnionParam{}, fmt.Errorf("file content requires file_url, file_data, or file_id")
			}
			return responses.ResponseInputContentUnionParam{
				OfInputFile: &fileParam,
			}, nil
		default:
			raw, err := json.Marshal(v)
			if err != nil {
				return responses.ResponseInputContentUnionParam{}, fmt.Errorf("marshal custom content: %w", err)
			}
			var union responses.ResponseInputContentUnionParam
			if err := json.Unmarshal(raw, &union); err != nil {
				return responses.ResponseInputContentUnionParam{}, fmt.Errorf("decode custom content: %w", err)
			}
			if !inputContentUnionHasValue(union) {
				return responses.ResponseInputContentUnionParam{}, fmt.Errorf("custom content %q missing recognized payload", kind)
			}
			return union, nil
		}
	default:
		return responses.ResponseInputContentUnionParam{}, fmt.Errorf("unsupported content item type %T", value)
	}
}

func normalizeMessageRole(role string) (responses.EasyInputMessageRole, error) {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "", "user":
		return responses.EasyInputMessageRoleUser, nil
	case "assistant":
		return responses.EasyInputMessageRoleAssistant, nil
	case "system":
		return responses.EasyInputMessageRoleSystem, nil
	case "developer":
		return responses.EasyInputMessageRoleDeveloper, nil
	default:
		return "", fmt.Errorf("role %q not supported", role)
	}
}

func defaultString(value string, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func messageContentUnionHasValue(union responses.EasyInputMessageContentUnionParam) bool {
	if union.OfString.Valid() {
		return true
	}
	if len(union.OfInputItemContentList) > 0 {
		return true
	}
	return false
}

func inputContentUnionHasValue(union responses.ResponseInputContentUnionParam) bool {
	return union.OfInputText != nil ||
		union.OfInputImage != nil ||
		union.OfInputFile != nil ||
		union.OfInputAudio != nil
}
