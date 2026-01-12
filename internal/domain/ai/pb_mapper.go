package ai

import (
	"time"

	aiv1 "github.com/uniedit/server/api/pb/ai"
	"github.com/uniedit/server/internal/model"
	"google.golang.org/protobuf/types/known/structpb"
)

func rateLimitToModel(in *aiv1.RateLimitConfig) *model.AIRateLimitConfig {
	if in == nil {
		return nil
	}
	return &model.AIRateLimitConfig{
		RPM:        int(in.GetRpm()),
		TPM:        int(in.GetTpm()),
		DailyLimit: int(in.GetDailyLimit()),
	}
}

func providerFromModel(p *model.AIProvider) *aiv1.Provider {
	if p == nil {
		return nil
	}

	var options *structpb.Struct
	if p.Options != nil {
		if s, err := structpb.NewStruct(p.Options); err == nil {
			options = s
		}
	}

	return &aiv1.Provider{
		Id:        p.ID.String(),
		Name:      p.Name,
		Type:      string(p.Type),
		BaseUrl:   p.BaseURL,
		Enabled:   p.Enabled,
		Weight:    int32(p.Weight),
		Priority:  int32(p.Priority),
		RateLimit: rateLimitFromModel(p.RateLimit),
		Options:   options,
		CreatedAt: formatTime(p.CreatedAt),
		UpdatedAt: formatTime(p.UpdatedAt),
	}
}

func rateLimitFromModel(in *model.AIRateLimitConfig) *aiv1.RateLimitConfig {
	if in == nil {
		return nil
	}
	return &aiv1.RateLimitConfig{
		Rpm:        int32(in.RPM),
		Tpm:        int32(in.TPM),
		DailyLimit: int32(in.DailyLimit),
	}
}

func modelFromModel(m *model.AIModel) *aiv1.Model {
	if m == nil {
		return nil
	}

	var options *structpb.Struct
	if m.Options != nil {
		if s, err := structpb.NewStruct(m.Options); err == nil {
			options = s
		}
	}

	return &aiv1.Model{
		Id:              m.ID,
		ProviderId:      m.ProviderID.String(),
		Name:            m.Name,
		Capabilities:    append([]string(nil), m.Capabilities...),
		ContextWindow:   int32(m.ContextWindow),
		MaxOutputTokens: int32(m.MaxOutputTokens),
		InputCostPer_1K:  m.InputCostPer1K,
		OutputCostPer_1K: m.OutputCostPer1K,
		Options:         options,
		Enabled:         m.Enabled,
		CreatedAt:       formatTime(m.CreatedAt),
		UpdatedAt:       formatTime(m.UpdatedAt),
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func chatRequestToModel(in *aiv1.ChatRequest) (*model.AIChatRequest, error) {
	if in == nil {
		return &model.AIChatRequest{}, nil
	}

	out := &model.AIChatRequest{
		Model:     in.GetModel(),
		MaxTokens: int(in.GetMaxTokens()),
		Stop:      append([]string(nil), in.GetStop()...),
		Stream:    in.GetStream(),
		Messages:  make([]*model.AIChatMessage, 0, len(in.GetMessages())),
		Tools:     make([]*model.AITool, 0, len(in.GetTools())),
		Metadata:  nil,
	}

	if v := in.GetTemperature(); v != nil {
		val := v.GetValue()
		out.Temperature = &val
	}
	if v := in.GetTopP(); v != nil {
		val := v.GetValue()
		out.TopP = &val
	}
	if md := in.GetMetadata(); md != nil {
		out.Metadata = md.AsMap()
	}
	if tc := in.GetToolChoice(); tc != nil {
		out.ToolChoice = tc.AsInterface()
	}

	for _, m := range in.GetMessages() {
		msg, err := chatMessageToModel(m)
		if err != nil {
			return nil, err
		}
		if msg == nil {
			continue
		}
		out.Messages = append(out.Messages, msg)
	}

	for _, t := range in.GetTools() {
		tool, err := toolToModel(t)
		if err != nil {
			return nil, err
		}
		if tool == nil {
			continue
		}
		out.Tools = append(out.Tools, tool)
	}

	return out, nil
}

func chatMessageToModel(in *aiv1.ChatMessage) (*model.AIChatMessage, error) {
	if in == nil {
		return nil, nil
	}

	var content any
	if in.GetContent() != nil {
		content = in.GetContent().AsInterface()
	}

	out := &model.AIChatMessage{
		Role:       in.GetRole(),
		Content:    content,
		Name:       in.GetName(),
		ToolCallID: in.GetToolCallId(),
		ToolCalls:  make([]*model.AIToolCall, 0, len(in.GetToolCalls())),
	}

	for _, tc := range in.GetToolCalls() {
		out.ToolCalls = append(out.ToolCalls, toolCallToModel(tc))
	}
	return out, nil
}

func toolToModel(in *aiv1.Tool) (*model.AITool, error) {
	if in == nil {
		return nil, nil
	}

	out := &model.AITool{
		Type: in.GetType(),
	}
	if in.GetFunction() != nil {
		fn, err := functionToModel(in.GetFunction())
		if err != nil {
			return nil, err
		}
		out.Function = fn
	}
	return out, nil
}

func functionToModel(in *aiv1.FunctionDef) (*model.AIFunction, error) {
	if in == nil {
		return nil, nil
	}

	out := &model.AIFunction{
		Name:        in.GetName(),
		Description: in.GetDescription(),
		Parameters:  nil,
	}
	if params := in.GetParameters(); params != nil {
		out.Parameters = params.AsMap()
	}
	return out, nil
}

func toolCallToModel(in *aiv1.ToolCall) *model.AIToolCall {
	if in == nil {
		return nil
	}
	out := &model.AIToolCall{
		ID:   in.GetId(),
		Type: in.GetType(),
	}
	if in.GetFunction() != nil {
		out.Function = &model.AIFunctionCall{
			Name:      in.GetFunction().GetName(),
			Arguments: in.GetFunction().GetArguments(),
		}
	}
	return out
}

func chatResponseFromModel(in *model.AIChatResponse) (*aiv1.ChatResponse, error) {
	if in == nil {
		return nil, nil
	}

	msg, err := chatMessageFromModel(in.Message)
	if err != nil {
		return nil, err
	}

	return &aiv1.ChatResponse{
		Id:           in.ID,
		Model:        in.Model,
		Message:      msg,
		FinishReason: in.FinishReason,
		Usage:        usageFromModel(in.Usage),
		Routing:      routingFromModel(in.Routing),
	}, nil
}

func chatMessageFromModel(in *model.AIChatMessage) (*aiv1.ChatMessage, error) {
	if in == nil {
		return nil, nil
	}

	content, err := structpb.NewValue(in.Content)
	if err != nil {
		return nil, err
	}

	out := &aiv1.ChatMessage{
		Role:       in.Role,
		Content:    content,
		Name:       in.Name,
		ToolCallId: in.ToolCallID,
		ToolCalls:  make([]*aiv1.ToolCall, 0, len(in.ToolCalls)),
	}

	for _, tc := range in.ToolCalls {
		out.ToolCalls = append(out.ToolCalls, toolCallFromModel(tc))
	}
	return out, nil
}

func toolCallFromModel(in *model.AIToolCall) *aiv1.ToolCall {
	if in == nil {
		return nil
	}
	out := &aiv1.ToolCall{
		Id:   in.ID,
		Type: in.Type,
	}
	if in.Function != nil {
		out.Function = &aiv1.FunctionCall{
			Name:      in.Function.Name,
			Arguments: in.Function.Arguments,
		}
	}
	return out
}

func usageFromModel(in *model.AIUsage) *aiv1.Usage {
	if in == nil {
		return nil
	}
	return &aiv1.Usage{
		PromptTokens:     int32(in.PromptTokens),
		CompletionTokens: int32(in.CompletionTokens),
		TotalTokens:      int32(in.TotalTokens),
	}
}

func routingFromModel(in *model.AIRoutingInfo) *aiv1.RoutingInfo {
	if in == nil {
		return nil
	}
	return &aiv1.RoutingInfo{
		ProviderUsed: in.ProviderUsed,
		ModelUsed:    in.ModelUsed,
		LatencyMs:    in.LatencyMs,
		CostUsd:      in.CostUSD,
	}
}
