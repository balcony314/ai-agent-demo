// Package agent 实现了基于 ReAct 模式的 AI Agent 框架。
//
// # 核心概念
//
// ReAct = Reasoning + Acting，让 LLM 从"只能说"变成"能做"。
// Agent 通过 ReAct 循环协调 LLM（大脑）和 Tools（手脚）完成用户任务。
//
// # 架构
//
//	┌─────────────────────────────────────────────────┐
//	│  Agent (agent.go)                               │
//	│  ┌───────────────────────────────────────────┐  │
//	│  │  ReAct Loop:                              │  │
//	│  │  用户消息 → LLM → tool_calls?             │  │
//	│  │    ├─ 是 → 执行工具 → 结果喂回 LLM       │  │
//	│  │    └─ 否 → 返回最终回答                   │  │
//	│  └───────────────────────────────────────────┘  │
//	│         │                    │                  │
//	│         ▼                    ▼                  │
//	│  LLMClient (llm.go)   ToolRegistry (tools.go)  │
//	│  SkillRegistry (skill.go)                       │
//	└─────────────────────────────────────────────────┘
//
// # 关键类型
//
//   - [Message]: 对话消息，对齐 OpenAI chat 格式
//   - [Tool]: 工具定义 = JSON Schema + Execute 函数
//   - [LLMClient]: LLM 接口，支持 OpenAIClient 和 MockClient
//   - [Skill]: 预设角色配置（SystemPrompt + 工具列表）
//   - [Agent]: 编排核心，组合 LLM + Tools + Skills
//
// # 快速开始
//
//	llm := agent.NewMockClient()
//	config := agent.DefaultConfig()
//	a := agent.NewAgent(llm, config)
//	reply, err := a.Run("你好")
//
// # 扩展
//
//   - 添加工具：在 RegisterBuiltinTools() 中注册 [Tool]
//   - 添加技能：在 RegisterBuiltinSkills() 中注册 [Skill]
//   - 替换 LLM：实现 [LLMClient] 接口
package agent
