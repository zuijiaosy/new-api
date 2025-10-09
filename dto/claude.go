package dto

import (
	"encoding/json"
	"fmt"
	"one-api/common"
	"one-api/types"
	"strings"

	"github.com/gin-gonic/gin"
)

type ClaudeMetadata struct {
	UserId string `json:"user_id"`
}

type ClaudeMediaMessage struct {
	Type         string               `json:"type,omitempty"`
	Text         *string              `json:"text,omitempty"`
	Model        string               `json:"model,omitempty"`
	Source       *ClaudeMessageSource `json:"source,omitempty"`
	Usage        *ClaudeUsage         `json:"usage,omitempty"`
	StopReason   *string              `json:"stop_reason,omitempty"`
	PartialJson  *string              `json:"partial_json,omitempty"`
	Role         string               `json:"role,omitempty"`
	Thinking     string               `json:"thinking,omitempty"`
	Signature    string               `json:"signature,omitempty"`
	Delta        string               `json:"delta,omitempty"`
	CacheControl json.RawMessage      `json:"cache_control,omitempty"`
	// tool_calls
	Id        string `json:"id,omitempty"`
	Name      string `json:"name,omitempty"`
	Input     any    `json:"input,omitempty"`
	Content   any    `json:"content,omitempty"`
	ToolUseId string `json:"tool_use_id,omitempty"`
}

func (c *ClaudeMediaMessage) SetText(s string) {
	c.Text = &s
}

func (c *ClaudeMediaMessage) GetText() string {
	if c.Text == nil {
		return ""
	}
	return *c.Text
}

func (c *ClaudeMediaMessage) IsStringContent() bool {
	if c.Content == nil {
		return false
	}
	_, ok := c.Content.(string)
	if ok {
		return true
	}
	return false
}

func (c *ClaudeMediaMessage) GetStringContent() string {
	if c.Content == nil {
		return ""
	}
	switch c.Content.(type) {
	case string:
		return c.Content.(string)
	case []any:
		var contentStr string
		for _, contentItem := range c.Content.([]any) {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == ContentTypeText {
				if subStr, ok := contentMap["text"].(string); ok {
					contentStr += subStr
				}
			}
		}
		return contentStr
	}

	return ""
}

func (c *ClaudeMediaMessage) GetJsonRowString() string {
	jsonContent, _ := common.Marshal(c)
	return string(jsonContent)
}

func (c *ClaudeMediaMessage) SetContent(content any) {
	c.Content = content
}

func (c *ClaudeMediaMessage) ParseMediaContent() []ClaudeMediaMessage {
	mediaContent, _ := common.Any2Type[[]ClaudeMediaMessage](c.Content)
	return mediaContent
}

type ClaudeMessageSource struct {
	Type      string `json:"type"`
	MediaType string `json:"media_type,omitempty"`
	Data      any    `json:"data,omitempty"`
	Url       string `json:"url,omitempty"`
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`
}

func (c *ClaudeMessage) IsStringContent() bool {
	if c.Content == nil {
		return false
	}
	_, ok := c.Content.(string)
	return ok
}

func (c *ClaudeMessage) GetStringContent() string {
	if c.Content == nil {
		return ""
	}
	switch c.Content.(type) {
	case string:
		return c.Content.(string)
	case []any:
		var contentStr string
		for _, contentItem := range c.Content.([]any) {
			contentMap, ok := contentItem.(map[string]any)
			if !ok {
				continue
			}
			if contentMap["type"] == ContentTypeText {
				if subStr, ok := contentMap["text"].(string); ok {
					contentStr += subStr
				}
			}
		}
		return contentStr
	}

	return ""
}

func (c *ClaudeMessage) SetStringContent(content string) {
	c.Content = content
}

func (c *ClaudeMessage) ParseContent() ([]ClaudeMediaMessage, error) {
	return common.Any2Type[[]ClaudeMediaMessage](c.Content)
}

type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description,omitempty"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

type InputSchema struct {
	Type       string `json:"type"`
	Properties any    `json:"properties,omitempty"`
	Required   any    `json:"required,omitempty"`
}

type ClaudeWebSearchTool struct {
	Type         string                       `json:"type"`
	Name         string                       `json:"name"`
	MaxUses      int                          `json:"max_uses,omitempty"`
	UserLocation *ClaudeWebSearchUserLocation `json:"user_location,omitempty"`
}

type ClaudeWebSearchUserLocation struct {
	Type     string `json:"type"`
	Timezone string `json:"timezone,omitempty"`
	Country  string `json:"country,omitempty"`
	Region   string `json:"region,omitempty"`
	City     string `json:"city,omitempty"`
}

type ClaudeToolChoice struct {
	Type                   string `json:"type"`
	Name                   string `json:"name,omitempty"`
	DisableParallelToolUse bool   `json:"disable_parallel_tool_use,omitempty"`
}

type ClaudeRequest struct {
	Model             string          `json:"model"`
	Prompt            string          `json:"prompt,omitempty"`
	System            any             `json:"system,omitempty"`
	Messages          []ClaudeMessage `json:"messages,omitempty"`
	MaxTokens         uint            `json:"max_tokens,omitempty"`
	MaxTokensToSample uint            `json:"max_tokens_to_sample,omitempty"`
	StopSequences     []string        `json:"stop_sequences,omitempty"`
	Temperature       *float64        `json:"temperature,omitempty"`
	TopP              float64         `json:"top_p,omitempty"`
	TopK              int             `json:"top_k,omitempty"`
	Stream            bool            `json:"stream,omitempty"`
	Tools             any             `json:"tools,omitempty"`
	ContextManagement json.RawMessage `json:"context_management,omitempty"`
	ToolChoice        any             `json:"tool_choice,omitempty"`
	Thinking          *Thinking       `json:"thinking,omitempty"`
	McpServers        json.RawMessage `json:"mcp_servers,omitempty"`
	Metadata          json.RawMessage `json:"metadata,omitempty"`
	// 服务层级字段，用于指定 API 服务等级。允许透传可能导致实际计费高于预期，默认应过滤
	ServiceTier string `json:"service_tier,omitempty"`
}

func (c *ClaudeRequest) GetTokenCountMeta() *types.TokenCountMeta {
	var tokenCountMeta = types.TokenCountMeta{
		TokenType: types.TokenTypeTokenizer,
		MaxTokens: int(c.MaxTokens),
	}

	var texts = make([]string, 0)
	var fileMeta = make([]*types.FileMeta, 0)

	// system
	if c.System != nil {
		if c.IsStringSystem() {
			sys := c.GetStringSystem()
			if sys != "" {
				texts = append(texts, sys)
			}
		} else {
			systemMedia := c.ParseSystem()
			for _, media := range systemMedia {
				switch media.Type {
				case "text":
					texts = append(texts, media.GetText())
				case "image":
					if media.Source != nil {
						data := media.Source.Url
						if data == "" {
							data = common.Interface2String(media.Source.Data)
						}
						if data != "" {
							fileMeta = append(fileMeta, &types.FileMeta{FileType: types.FileTypeImage, OriginData: data})
						}
					}
				}
			}
		}
	}

	// messages
	for _, message := range c.Messages {
		tokenCountMeta.MessagesCount++
		texts = append(texts, message.Role)
		if message.IsStringContent() {
			content := message.GetStringContent()
			if content != "" {
				texts = append(texts, content)
			}
			continue
		}

		content, _ := message.ParseContent()
		for _, media := range content {
			switch media.Type {
			case "text":
				texts = append(texts, media.GetText())
			case "image":
				if media.Source != nil {
					data := media.Source.Url
					if data == "" {
						data = common.Interface2String(media.Source.Data)
					}
					if data != "" {
						fileMeta = append(fileMeta, &types.FileMeta{FileType: types.FileTypeImage, OriginData: data})
					}
				}
			case "tool_use":
				if media.Name != "" {
					texts = append(texts, media.Name)
				}
				if media.Input != nil {
					b, _ := common.Marshal(media.Input)
					texts = append(texts, string(b))
				}
			case "tool_result":
				if media.Content != nil {
					b, _ := common.Marshal(media.Content)
					texts = append(texts, string(b))
				}
			}
		}
	}

	// tools
	if c.Tools != nil {
		tools := c.GetTools()
		normalTools, webSearchTools := ProcessTools(tools)
		if normalTools != nil {
			for _, t := range normalTools {
				tokenCountMeta.ToolsCount++
				if t.Name != "" {
					texts = append(texts, t.Name)
				}
				if t.Description != "" {
					texts = append(texts, t.Description)
				}
				if t.InputSchema != nil {
					b, _ := common.Marshal(t.InputSchema)
					texts = append(texts, string(b))
				}
			}
		}
		if webSearchTools != nil {
			for _, t := range webSearchTools {
				tokenCountMeta.ToolsCount++
				if t.Name != "" {
					texts = append(texts, t.Name)
				}
				if t.UserLocation != nil {
					b, _ := common.Marshal(t.UserLocation)
					texts = append(texts, string(b))
				}
			}
		}
	}

	tokenCountMeta.CombineText = strings.Join(texts, "\n")
	tokenCountMeta.Files = fileMeta
	return &tokenCountMeta
}

func (c *ClaudeRequest) IsStream(ctx *gin.Context) bool {
	return c.Stream
}

func (c *ClaudeRequest) SetModelName(modelName string) {
	if modelName != "" {
		c.Model = modelName
	}
}

func (c *ClaudeRequest) SearchToolNameByToolCallId(toolCallId string) string {
	for _, message := range c.Messages {
		content, _ := message.ParseContent()
		for _, mediaMessage := range content {
			if mediaMessage.Id == toolCallId {
				return mediaMessage.Name
			}
		}
	}
	return ""
}

// AddTool 添加工具到请求中
func (c *ClaudeRequest) AddTool(tool any) {
	if c.Tools == nil {
		c.Tools = make([]any, 0)
	}

	switch tools := c.Tools.(type) {
	case []any:
		c.Tools = append(tools, tool)
	default:
		// 如果Tools不是[]any类型，重新初始化为[]any
		c.Tools = []any{tool}
	}
}

// GetTools 获取工具列表
func (c *ClaudeRequest) GetTools() []any {
	if c.Tools == nil {
		return nil
	}

	switch tools := c.Tools.(type) {
	case []any:
		return tools
	default:
		return nil
	}
}

// ProcessTools 处理工具列表，支持类型断言
func ProcessTools(tools []any) ([]*Tool, []*ClaudeWebSearchTool) {
	var normalTools []*Tool
	var webSearchTools []*ClaudeWebSearchTool

	for _, tool := range tools {
		switch t := tool.(type) {
		case *Tool:
			normalTools = append(normalTools, t)
		case *ClaudeWebSearchTool:
			webSearchTools = append(webSearchTools, t)
		case Tool:
			normalTools = append(normalTools, &t)
		case ClaudeWebSearchTool:
			webSearchTools = append(webSearchTools, &t)
		default:
			// 未知类型，跳过
			continue
		}
	}

	return normalTools, webSearchTools
}

type Thinking struct {
	Type         string `json:"type"`
	BudgetTokens *int   `json:"budget_tokens,omitempty"`
}

func (c *Thinking) GetBudgetTokens() int {
	if c.BudgetTokens == nil {
		return 0
	}
	return *c.BudgetTokens
}

func (c *ClaudeRequest) IsStringSystem() bool {
	_, ok := c.System.(string)
	return ok
}

func (c *ClaudeRequest) GetStringSystem() string {
	if c.IsStringSystem() {
		return c.System.(string)
	}
	return ""
}

func (c *ClaudeRequest) SetStringSystem(system string) {
	c.System = system
}

func (c *ClaudeRequest) ParseSystem() []ClaudeMediaMessage {
	mediaContent, _ := common.Any2Type[[]ClaudeMediaMessage](c.System)
	return mediaContent
}

type ClaudeErrorWithStatusCode struct {
	Error      types.ClaudeError `json:"error"`
	StatusCode int               `json:"status_code"`
	LocalError bool
}

type ClaudeResponse struct {
	Id           string               `json:"id,omitempty"`
	Type         string               `json:"type"`
	Role         string               `json:"role,omitempty"`
	Content      []ClaudeMediaMessage `json:"content,omitempty"`
	Completion   string               `json:"completion,omitempty"`
	StopReason   string               `json:"stop_reason,omitempty"`
	Model        string               `json:"model,omitempty"`
	Error        any                  `json:"error,omitempty"`
	Usage        *ClaudeUsage         `json:"usage,omitempty"`
	Index        *int                 `json:"index,omitempty"`
	ContentBlock *ClaudeMediaMessage  `json:"content_block,omitempty"`
	Delta        *ClaudeMediaMessage  `json:"delta,omitempty"`
	Message      *ClaudeMediaMessage  `json:"message,omitempty"`
}

// set index
func (c *ClaudeResponse) SetIndex(i int) {
	c.Index = &i
}

// get index
func (c *ClaudeResponse) GetIndex() int {
	if c.Index == nil {
		return 0
	}
	return *c.Index
}

// GetClaudeError 从动态错误类型中提取ClaudeError结构
func (c *ClaudeResponse) GetClaudeError() *types.ClaudeError {
	if c.Error == nil {
		return nil
	}

	switch err := c.Error.(type) {
	case types.ClaudeError:
		return &err
	case *types.ClaudeError:
		return err
	case map[string]interface{}:
		// 处理从JSON解析来的map结构
		claudeErr := &types.ClaudeError{}
		if errType, ok := err["type"].(string); ok {
			claudeErr.Type = errType
		}
		if errMsg, ok := err["message"].(string); ok {
			claudeErr.Message = errMsg
		}
		return claudeErr
	case string:
		// 处理简单字符串错误
		return &types.ClaudeError{
			Type:    "upstream_error",
			Message: err,
		}
	default:
		// 未知类型，尝试转换为字符串
		return &types.ClaudeError{
			Type:    "unknown_upstream_error",
			Message: fmt.Sprintf("unknown_error: %v", err),
		}
	}
}

type ClaudeUsage struct {
	InputTokens              int                  `json:"input_tokens"`
	CacheCreationInputTokens int                  `json:"cache_creation_input_tokens"`
	CacheReadInputTokens     int                  `json:"cache_read_input_tokens"`
	OutputTokens             int                  `json:"output_tokens"`
	ServerToolUse            *ClaudeServerToolUse `json:"server_tool_use,omitempty"`
}

type ClaudeServerToolUse struct {
	WebSearchRequests int `json:"web_search_requests"`
}
