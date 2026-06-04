package agent

// ═══════════════════════════════════════════════════════════════
// agent.go — Agent 核心：ReAct 循环
// ═══════════════════════════════════════════════════════════════
//
// 【教学要点】什么是 ReAct？
//
// ReAct = Reasoning + Acting（推理 + 行动）
//
// 传统 LLM 只能 "想"（生成文本），不能 "做"（执行操作）。
// ReAct 让 LLM 变成 Agent：
//
//   ┌─────────────────────────────────────────────────┐
//   │                                                 │
//   │  User: "北京现在几点？10分钟前呢？"              │
//   │                                                 │
//   │  ┌─── ReAct Loop ───────────────────────────┐   │
//   │  │                                          │   │
//   │  │  THINK: 用户问时间，我需要先获取当前时间   │   │
//   │  │    → 决定调用 current_time 工具           │   │
//   │  │                                          │   │
//   │  │  ACT: 调用 current_time()               │   │
//   │  │    → 返回 "14:30:25"                     │   │
//   │  │                                          │   │
//   │  │  OBSERVE: 当前是 14:30                    │   │
//   │  │    → 10分钟前是 14:20                    │   │
//   │  │                                          │   │
//   │  │  THINK: 信息够了，可以回答了              │   │
//   │  │    → 生成最终回复                        │   │
//   │  │                                          │   │
//   │  └──────────────────────────────────────────┘   │
//   │                                                 │
//   │  Agent: "现在是 14:30，10分钟前是 14:20"        │
//   │                                                 │
//   └─────────────────────────────────────────────────┘
//
// 关键洞察：LLM 不直接输出答案，而是输出 "我想要做什么"（tool_calls），
// Agent 框架执行后把结果喂回去，LLM 再决定下一步。
// 这个循环可以执行多轮，直到 LLM 认为信息足够了。

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Agent 是一个 AI 智能体
type Agent struct {
	config       Config          // 配置
	llm          LLMClient      // LLM 客户端
	registry     *ToolRegistry  // 工具注册表
	skillReg     *SkillRegistry // 技能注册表
	currentSkill string         // 当前激活的技能名称
	history      []Message      // 对话历史
}

// NewAgent 创建一个新的 Agent
func NewAgent(llm LLMClient, config Config) *Agent {
	// 注册内置工具
	registry := NewToolRegistry()
	RegisterBuiltinTools(registry)

	// 注册内置技能
	skillReg := NewSkillRegistry()
	RegisterBuiltinSkills(skillReg)

	// 初始化对话历史（system prompt 是第一条消息）
	history := []Message{
		{Role: RoleSystem, Content: config.SystemPrompt},
	}

	return &Agent{
		config:       config,
		llm:          llm,
		registry:     registry,
		skillReg:     skillReg,
		currentSkill: "general", // 默认使用通用技能
		history:      history,
	}
}

// ─── 核心：ReAct 循环 ──────────────────────────────────────────

// Run 是 Agent 的主循环
// 每次用户输入一条消息，Agent 会：
//   1. 把消息加入对话历史
//   2. 进入 ReAct 循环（最多 MaxTurns 轮）
//   3. 每轮让 LLM 决定下一步（回复 or 调用工具）
//   4. 如果 LLM 要调用工具 → 执行工具 → 把结果放回历史 → 继续循环
//   5. 如果 LLM 直接回复 → 返回最终答案 → 退出循环
func (a *Agent) Run(userInput string) (string, error) {
	fmt.Printf("\n%s\n", strings.Repeat("─", 50))
	fmt.Printf("📝 用户: %s\n", userInput)
	fmt.Printf("%s\n\n", strings.Repeat("─", 50))

	// 1. 把用户消息加入历史
	a.history = append(a.history, Message{
		Role:    RoleUser,
		Content: userInput,
	})

	// 2. ReAct 循环：最多 MaxTurns 轮，每轮 LLM 要么调用工具，要么给出最终回答
	for turn := 0; turn < a.config.MaxTurns; turn++ {
		fmt.Printf("🔄 推理轮次 %d/%d\n", turn+1, a.config.MaxTurns)

		// 3. 把当前对话历史 + 工具定义发给 LLM
		//    LLM 会根据历史和工具描述，决定是回复文本还是调用工具
		//    根据当前技能过滤可用工具
		tools := a.getSkillTools()
		response, err := a.llm.Chat(a.history, tools)
		if err != nil {
			return "", fmt.Errorf("LLM 调用失败: %w", err)
		}

		// 4. 把 LLM 的回复加入历史
		a.history = append(a.history, *response)

		// 5. 判断 LLM 是否要使用工具
		if len(response.ToolCalls) == 0 {
			// 没有工具调用 → LLM 认为可以直接回答了
			fmt.Printf("✅ Agent 回复完成 (共 %d 轮推理)\n\n", turn+1)
			return response.Content, nil
		}

		// 6. 有工具调用 → 执行每个工具，把结果作为 RoleTool 消息加入历史
		//    这就是 ReAct 中的 "Act" 和 "Observe"：
		//    Act = 执行工具，Observe = 把结果喂回 LLM 让它继续推理
		fmt.Printf("🛠️  LLM 请求调用 %d 个工具:\n", len(response.ToolCalls))

		for _, tc := range response.ToolCalls {
			toolName := tc.Function.Name
			toolArgs := tc.Function.Arguments // JSON 字符串，需要解析后传给工具

			fmt.Printf("   → 调用 %s(%s)\n", toolName, TruncStr(toolArgs, 60))

			// 根据工具名从注册表查找并执行
			result, err := a.executeTool(toolName, toolArgs)

			// 构建工具结果消息
			var resultContent string
			if err != nil {
				resultContent = fmt.Sprintf("工具执行错误: %v", err)
				fmt.Printf("   ❌ 错误: %v\n", err)
			} else {
				resultContent = result
				fmt.Printf("   📋 结果: %s\n", TruncStr(result, 80))
			}

			// 把工具结果加入对话历史
			a.history = append(a.history, Message{
				Role:       RoleTool,
				Content:    resultContent,
				ToolCallID: tc.ID,
			})
		}

		fmt.Println() // 空行分隔轮次
	}

	// 超过最大轮数
	return "抱歉，推理轮次已达上限。", fmt.Errorf("达到最大推理轮数: %d", a.config.MaxTurns)
}

// executeTool 执行一个工具
func (a *Agent) executeTool(name, argsJSON string) (string, error) {
	tool, ok := a.registry.Get(name)
	if !ok {
		return "", fmt.Errorf("未知工具: %s", name)
	}

	var args json.RawMessage = []byte(argsJSON)
	result, err := tool.Execute(args)
	if err != nil {
		return "", err
	}

	return result, nil
}

// ─── 辅助函数 ──────────────────────────────────────────────────

// GetHistory 获取对话历史（用于调试）
func (a *Agent) GetHistory() []Message {
	return a.history
}

// ClearHistory 清空对话历史（保留 system prompt）
func (a *Agent) ClearHistory() {
	a.history = a.history[:1]
}

// ListTools 列出当前技能可用的工具
func (a *Agent) ListTools() []string {
	skill, ok := a.skillReg.Get(a.currentSkill)
	if !ok || len(skill.Tools) == 0 {
		return a.registry.Names()
	}
	return skill.Tools
}

// ─── Skill 相关方法 ────────────────────────────────────────────

// SwitchSkill 切换当前技能
func (a *Agent) SwitchSkill(name string) error {
	skill, ok := a.skillReg.Get(name)
	if !ok {
		return fmt.Errorf("未知技能: %s", name)
	}

	a.currentSkill = name

	// 更新 system prompt
	a.history[0] = Message{
		Role:    RoleSystem,
		Content: skill.SystemPrompt,
	}

	// 清空对话历史（保留新的 system prompt）
	a.ClearHistory()

	fmt.Printf("✨ 已切换到技能: %s (%s)\n", name, skill.Description)
	return nil
}

// CurrentSkill 获取当前技能名称
func (a *Agent) CurrentSkill() string {
	return a.currentSkill
}

// ListSkills 列出所有可用技能
func (a *Agent) ListSkills() []Skill {
	return a.skillReg.List()
}

// FormatSkillList 格式化技能列表
func (a *Agent) FormatSkillList() string {
	return FormatSkillList(a.skillReg.List(), a.currentSkill)
}

// getSkillTools 根据当前技能获取可用工具定义
func (a *Agent) getSkillTools() []ToolDefinition {
	skill, ok := a.skillReg.Get(a.currentSkill)
	if !ok || len(skill.Tools) == 0 {
		// 未指定工具列表 → 使用全部工具
		return a.registry.Definitions()
	}

	// 按技能配置过滤工具
	defs := make([]ToolDefinition, 0, len(skill.Tools))
	for _, name := range skill.Tools {
		if tool, found := a.registry.Get(name); found {
			defs = append(defs, tool.Definition)
		}
	}
	return defs
}

// TruncStr 截断字符串用于日志显示
// 工具参数和结果可能很长，日志中只显示前 maxLen 个字符
func TruncStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
