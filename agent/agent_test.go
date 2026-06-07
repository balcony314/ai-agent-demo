package agent

import (
	"fmt"
	"strings"
	"testing"
)

// ─── Mock LLMClient（用于测试 Agent.Run）──────────────────────

// scriptedClient 按预设脚本返回响应的 mock LLM
type scriptedClient struct {
	responses []*Message // 预设的响应队列
	callIdx   int        // 当前调用索引
}

func (c *scriptedClient) Chat(messages []Message, tools []ToolDefinition) (*Message, error) {
	if c.callIdx >= len(c.responses) {
		return nil, fmt.Errorf("scriptedClient: 预设响应已用完 (callIdx=%d)", c.callIdx)
	}
	resp := c.responses[c.callIdx]
	c.callIdx++
	return resp, nil
}

// errorClient 总是返回错误的 mock LLM
type errorClient struct{}

func (c *errorClient) Chat(messages []Message, tools []ToolDefinition) (*Message, error) {
	return nil, fmt.Errorf("模拟 LLM 错误")
}

// ─── Agent.Run 测试 ───────────────────────────────────────────

func TestAgentRun_DirectReply(t *testing.T) {
	// LLM 直接回复文本，不调用工具
	client := &scriptedClient{
		responses: []*Message{
			{Role: RoleAssistant, Content: "你好！"},
		},
	}

	agent := NewAgent(client, DefaultConfig())
	result, err := agent.Run("你好")
	if err != nil {
		t.Fatalf("Run 错误: %v", err)
	}
	if result != "你好！" {
		t.Errorf("结果 = %q, 期望 %q", result, "你好！")
	}
}

func TestAgentRun_ToolCallThenReply(t *testing.T) {
	// 第一轮 LLM 调用 search 工具，第二轮 LLM 根据结果回复
	client := &scriptedClient{
		responses: []*Message{
			// 第一轮：LLM 决定调用 search
			{
				Role:    RoleAssistant,
				Content: "",
				ToolCalls: []ToolCall{
					{
						ID:   "call_1",
						Type: "function",
						Function: FunctionCall{
							Name:      "search",
							Arguments: `{"query":"golang"}`,
						},
					},
				},
			},
			// 第二轮：LLM 看到工具结果后回复
			{Role: RoleAssistant, Content: "Go 是 Google 开发的编程语言。"},
		},
	}

	agent := NewAgent(client, DefaultConfig())
	result, err := agent.Run("什么是 Go 语言？")
	if err != nil {
		t.Fatalf("Run 错误: %v", err)
	}
	if result != "Go 是 Google 开发的编程语言。" {
		t.Errorf("结果 = %q, 期望 %q", result, "Go 是 Google 开发的编程语言。")
	}

	// 验证对话历史：system + user + assistant(tool_call) + tool_result + assistant(reply)
	history := agent.GetHistory()
	if len(history) != 5 {
		t.Errorf("历史长度 = %d, 期望 5", len(history))
	}
	// 第3条应是 tool result
	if history[3].Role != RoleTool {
		t.Errorf("历史[3] role = %q, 期望 %q", history[3].Role, RoleTool)
	}
	if history[3].ToolCallID != "call_1" {
		t.Errorf("历史[3] ToolCallID = %q, 期望 %q", history[3].ToolCallID, "call_1")
	}
}

func TestAgentRun_MultipleToolCalls(t *testing.T) {
	// LLM 连续调用多个工具
	client := &scriptedClient{
		responses: []*Message{
			// 第一轮：调用 calculator
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID:   "call_calc",
						Type: "function",
						Function: FunctionCall{
							Name:      "calculator",
							Arguments: `{"expression":"2+3"}`,
						},
					},
				},
			},
			// 第二轮：调用 search
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID:   "call_search",
						Type: "function",
						Function: FunctionCall{
							Name:      "search",
							Arguments: `{"query":"agent"}`,
						},
					},
				},
			},
			// 第三轮：最终回复
			{Role: RoleAssistant, Content: "计算和搜索都完成了。"},
		},
	}

	agent := NewAgent(client, DefaultConfig())
	result, err := agent.Run("帮我算 2+3 并搜索 agent")
	if err != nil {
		t.Fatalf("Run 错误: %v", err)
	}
	if result != "计算和搜索都完成了。" {
		t.Errorf("结果 = %q", result)
	}
}

func TestAgentRun_LLMError(t *testing.T) {
	client := &errorClient{}
	agent := NewAgent(client, DefaultConfig())

	_, err := agent.Run("你好")
	if err == nil {
		t.Fatal("LLM 错误时 Run 应返回错误")
	}
	if !strings.Contains(err.Error(), "LLM 调用失败") {
		t.Errorf("错误信息 = %q, 期望包含 'LLM 调用失败'", err.Error())
	}
}

func TestAgentRun_UnknownTool(t *testing.T) {
	// LLM 调用一个不存在的工具
	client := &scriptedClient{
		responses: []*Message{
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID:   "call_bad",
						Type: "function",
						Function: FunctionCall{
							Name:      "nonexistent_tool",
							Arguments: `{}`,
						},
					},
				},
			},
			// Agent 会把工具错误喂回 LLM，LLM 再回复
			{Role: RoleAssistant, Content: "抱歉，工具调用失败了。"},
		},
	}

	agent := NewAgent(client, DefaultConfig())
	result, err := agent.Run("测试")
	if err != nil {
		t.Fatalf("Run 错误: %v", err)
	}

	// 工具错误应被记录到历史中，LLM 收到后给出最终回复
	history := agent.GetHistory()
	toolMsg := history[3] // system + user + assistant(tool_call) + tool_result
	if toolMsg.Role != RoleTool {
		t.Errorf("工具结果角色 = %q, 期望 %q", toolMsg.Role, RoleTool)
	}
	if !strings.Contains(toolMsg.Content, "未知工具") {
		t.Errorf("工具错误信息 = %q, 期望包含 '未知工具'", toolMsg.Content)
	}
	_ = result
}

func TestAgentRun_MaxTurnsExceeded(t *testing.T) {
	// LLM 每轮都调用工具，永不给出最终回复
	client := &scriptedClient{
		responses: func() []*Message {
			msgs := make([]*Message, 20) // 足够多的工具调用
			for i := range msgs {
				msgs[i] = &Message{
					Role: RoleAssistant,
					ToolCalls: []ToolCall{
						{
							ID:   fmt.Sprintf("call_%d", i),
							Type: "function",
							Function: FunctionCall{
								Name:      "calculator",
								Arguments: `{"expression":"1+1"}`,
							},
						},
					},
				}
			}
			return msgs
		}(),
	}

	config := DefaultConfig()
	config.MaxTurns = 3
	agent := NewAgent(client, config)

	result, err := agent.Run("无限循环测试")
	if err == nil {
		t.Fatal("超过最大轮数应返回错误")
	}
	if !strings.Contains(err.Error(), "最大推理轮数") {
		t.Errorf("错误 = %q, 期望包含 '最大推理轮数'", err.Error())
	}
	if result != "抱歉，推理轮次已达上限。" {
		t.Errorf("结果 = %q, 期望超限提示", result)
	}
}

// ─── Agent.executeTool 测试 ───────────────────────────────────

func TestAgentExecuteTool(t *testing.T) {
	client := &scriptedClient{}
	agent := NewAgent(client, DefaultConfig())

	// 正常执行
	result, err := agent.executeTool("calculator", `{"expression":"2+3"}`)
	if err != nil {
		t.Fatalf("executeTool 错误: %v", err)
	}
	if result == "" {
		t.Error("结果不应为空")
	}

	// 未知工具
	_, err = agent.executeTool("nonexistent", `{}`)
	if err == nil {
		t.Error("未知工具应返回错误")
	}
}

// ─── Agent.SwitchSkill 测试 ───────────────────────────────────

func TestAgentSwitchSkill(t *testing.T) {
	client := &scriptedClient{}
	agent := NewAgent(client, DefaultConfig())

	if agent.CurrentSkill() != "general" {
		t.Fatalf("默认技能 = %q, 期望 %q", agent.CurrentSkill(), "general")
	}

	// 切换到 coder
	err := agent.SwitchSkill("coder")
	if err != nil {
		t.Fatalf("SwitchSkill 错误: %v", err)
	}
	if agent.CurrentSkill() != "coder" {
		t.Errorf("当前技能 = %q, 期望 %q", agent.CurrentSkill(), "coder")
	}

	// 未知技能
	err = agent.SwitchSkill("nonexistent")
	if err == nil {
		t.Error("切换到未知技能应返回错误")
	}
}

func TestAgentSwitchSkill_ClearsHistory(t *testing.T) {
	client := &scriptedClient{
		responses: []*Message{
			{Role: RoleAssistant, Content: "回复"},
		},
	}
	agent := NewAgent(client, DefaultConfig())

	// 先对话一轮
	agent.Run("你好")
	before := len(agent.GetHistory())
	if before < 2 {
		t.Fatalf("对话后历史长度 = %d, 期望 >= 2", before)
	}

	// 切换技能应清空历史
	agent.SwitchSkill("coder")
	after := len(agent.GetHistory())
	if after != 1 {
		t.Errorf("切换技能后历史长度 = %d, 期望 1 (仅 system prompt)", after)
	}
}

// ─── Agent.getSkillTools 测试 ─────────────────────────────────

func TestAgentGetSkillTools_AllTools(t *testing.T) {
	client := &scriptedClient{}
	agent := NewAgent(client, DefaultConfig())

	// general 技能：使用全部工具
	defs := agent.getSkillTools()
	if len(defs) != 4 {
		t.Errorf("general 技能工具数 = %d, 期望 4", len(defs))
	}
}

func TestAgentGetSkillTools_FilteredTools(t *testing.T) {
	client := &scriptedClient{}
	agent := NewAgent(client, DefaultConfig())

	agent.SwitchSkill("coder") // 只有 calculator, text_transform
	defs := agent.getSkillTools()
	if len(defs) != 2 {
		t.Errorf("coder 技能工具数 = %d, 期望 2", len(defs))
	}

	names := map[string]bool{}
	for _, d := range defs {
		names[d.Function.Name] = true
	}
	if !names["calculator"] || !names["text_transform"] {
		t.Errorf("coder 工具 = %v, 期望 calculator 和 text_transform", names)
	}
}

// ─── Agent.ClearHistory 测试 ──────────────────────────────────

func TestAgentClearHistory(t *testing.T) {
	client := &scriptedClient{
		responses: []*Message{
			{Role: RoleAssistant, Content: "回复"},
		},
	}
	agent := NewAgent(client, DefaultConfig())
	agent.Run("你好")

	if len(agent.GetHistory()) < 2 {
		t.Fatal("对话后历史应有多条消息")
	}

	agent.ClearHistory()
	if len(agent.GetHistory()) != 1 {
		t.Errorf("清空后历史长度 = %d, 期望 1", len(agent.GetHistory()))
	}
	if agent.GetHistory()[0].Role != RoleSystem {
		t.Errorf("清空后第一条消息角色 = %q, 期望 %q", agent.GetHistory()[0].Role, RoleSystem)
	}
}

// ─── Agent.ListTools 测试 ─────────────────────────────────────

func TestAgentListTools(t *testing.T) {
	client := &scriptedClient{}
	agent := NewAgent(client, DefaultConfig())

	tools := agent.ListTools()
	if len(tools) == 0 {
		t.Error("ListTools 不应返回空")
	}
}

// ─── Agent.ListSkills 测试 ────────────────────────────────────

func TestAgentListSkills(t *testing.T) {
	client := &scriptedClient{}
	agent := NewAgent(client, DefaultConfig())

	skills := agent.ListSkills()
	if len(skills) != 5 {
		t.Errorf("ListSkills 长度 = %d, 期望 5", len(skills))
	}
}

// ─── MockClient 测试 ─────────────────────────────────────────

func TestMockClient_DirectReply(t *testing.T) {
	client := NewMockClient()
	messages := []Message{
		{Role: RoleSystem, Content: "你是助手"},
		{Role: RoleUser, Content: "你好"},
	}

	resp, err := client.Chat(messages, nil)
	if err != nil {
		t.Fatalf("Chat 错误: %v", err)
	}
	if resp.Role != RoleAssistant {
		t.Errorf("角色 = %q, 期望 %q", resp.Role, RoleAssistant)
	}
	if resp.Content == "" {
		t.Error("回复内容不应为空")
	}
}

func TestMockClient_ToolCall(t *testing.T) {
	client := NewMockClient()
	messages := []Message{
		{Role: RoleSystem, Content: "你是助手"},
		{Role: RoleUser, Content: "搜索一下"},
	}
	tools := []ToolDefinition{
		{Type: "function", Function: FunctionSchema{Name: "search"}},
	}

	resp, err := client.Chat(messages, tools)
	if err != nil {
		t.Fatalf("Chat 错误: %v", err)
	}
	if len(resp.ToolCalls) == 0 {
		t.Error("有工具时应返回 tool_calls")
	}
}

func TestMockClient_SummarizeToolResult(t *testing.T) {
	client := NewMockClient()
	messages := []Message{
		{Role: RoleSystem, Content: "你是助手"},
		{Role: RoleUser, Content: "搜索"},
		{Role: RoleTool, Content: "搜索结果内容"},
	}

	resp, err := client.Chat(messages, nil)
	if err != nil {
		t.Fatalf("Chat 错误: %v", err)
	}
	if !strings.Contains(resp.Content, "工具返回的结果") {
		t.Errorf("总结回复 = %q, 期望包含 '工具返回的结果'", resp.Content)
	}
}

// ─── NewAgent 默认配置测试 ────────────────────────────────────

func TestNewAgent_DefaultConfig(t *testing.T) {
	client := &scriptedClient{}
	config := DefaultConfig()
	agent := NewAgent(client, config)

	if agent.CurrentSkill() != "general" {
		t.Errorf("默认技能 = %q, 期望 %q", agent.CurrentSkill(), "general")
	}

	history := agent.GetHistory()
	if len(history) != 1 {
		t.Errorf("初始历史长度 = %d, 期望 1", len(history))
	}
	if history[0].Role != RoleSystem {
		t.Errorf("初始消息角色 = %q, 期望 %q", history[0].Role, RoleSystem)
	}
}

// ─── 边界情况：空工具调用参数 ─────────────────────────────────

func TestAgentRun_EmptyToolArgs(t *testing.T) {
	client := &scriptedClient{
		responses: []*Message{
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID:       "call_1",
						Type:     "function",
						Function: FunctionCall{Name: "current_time", Arguments: `{}`},
					},
				},
			},
			{Role: RoleAssistant, Content: "现在时间是..."},
		},
	}

	agent := NewAgent(client, DefaultConfig())
	result, err := agent.Run("几点了")
	if err != nil {
		t.Fatalf("Run 错误: %v", err)
	}
	if result == "" {
		t.Error("结果不应为空")
	}
}

// ─── 边界情况：多个并行工具调用 ──────────────────────────────

func TestAgentRun_ParallelToolCalls(t *testing.T) {
	client := &scriptedClient{
		responses: []*Message{
			// LLM 一次返回多个 tool_calls
			{
				Role: RoleAssistant,
				ToolCalls: []ToolCall{
					{
						ID: "call_1", Type: "function",
						Function: FunctionCall{Name: "calculator", Arguments: `{"expression":"1+1"}`},
					},
					{
						ID: "call_2", Type: "function",
						Function: FunctionCall{Name: "calculator", Arguments: `{"expression":"2+2"}`},
					},
				},
			},
			{Role: RoleAssistant, Content: "1+1=2, 2+2=4"},
		},
	}

	agent := NewAgent(client, DefaultConfig())
	result, err := agent.Run("算一下 1+1 和 2+2")
	if err != nil {
		t.Fatalf("Run 错误: %v", err)
	}

	// 历史中应有 2 条 tool result
	history := agent.GetHistory()
	toolResults := 0
	for _, m := range history {
		if m.Role == RoleTool {
			toolResults++
		}
	}
	if toolResults != 2 {
		t.Errorf("工具结果数 = %d, 期望 2", toolResults)
	}
	_ = result
}

// ─── DefaultConfig 测试 ──────────────────────────────────────

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	if config.MaxTurns != 10 {
		t.Errorf("MaxTurns = %d, 期望 10", config.MaxTurns)
	}
	if config.SystemPrompt == "" {
		t.Error("SystemPrompt 不应为空")
	}
}
