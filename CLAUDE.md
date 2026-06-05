# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本仓库中工作提供指引。

## 项目概述

用 Go 实现的教学级 AI Agent，演示 **ReAct (Reasoning + Acting)** 模式。通过 OpenAI 兼容的 function calling 实现带工具调用（计算器、时间、搜索、文本处理）的对话式 Agent。使用 cobra 管理 CLI 命令。

## 常用命令

```bash
go run .                                          # 交互式 REPL（Mock 模式）
go run . --api-key sk-xxx --model gpt-4o          # 接 OpenAI
go run . --api-key dummy -u http://localhost:11434/v1 -m qwen2  # 接 Ollama
go run . chat "你好"                              # 单次提问
go run . skill list                               # 查看技能列表
go run . version                                  # 版本信息
go build -o ai-agent .                            # 编译
go test ./...                                     # 测试（目前无测试文件）
go vet ./...                                      # 静态检查
```

全局 flags：`--api-key`/`-k`、`--base-url`/`-u`、`--model`/`-m`（默认 `gpt-4o`）、`--mock`

## 架构

核心是 ReAct 循环：LLM 生成文本或工具调用 → Agent 执行工具 → 把结果喂回 LLM → 循环（最多 10 轮），直到 LLM 输出最终文本回答。

```
main.go           → 入口：调用 cmd.Execute()
cmd/
  root.go         → Root 命令 + 全局 flags + 交互式 REPL
  chat.go         → chat 子命令：单次提问（非交互式）
  version.go      → version 子命令：版本信息
agent/
  types.go        → 核心类型：Message, ToolCall, ToolDefinition, Config, LLMClient 接口
  tools.go        → ToolRegistry（map 实现）+ RegisterBuiltinTools() 注册 4 个内置工具
  skill.go        → SkillRegistry + RegisterBuiltinSkills() 注册 5 个内置技能
  llm.go          → OpenAIClient（真实 /v1/chat/completions）+ MockClient（演示用）
  agent.go        → Agent.Run() —— ReAct 编排循环 + Skill 切换
```

关键抽象：
- **`LLMClient` 接口**（`Chat` 方法）—— 可在 `OpenAIClient` 和 `MockClient` 之间切换
- **`ToolRegistry`** —— `map[string]Tool`，提供 `Register`/`Get`/`Definitions`；每个 `Tool` 由 JSON Schema 的 `ToolDefinition` + `Execute` 函数组成
- **`Message`** —— role + content + 可选的 tool_calls/tool_call_id，对齐 OpenAI chat 格式

## 添加新工具

在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中注册。提供带 JSON Schema 参数的 `ToolDefinition` 和一个 `Execute` 函数。LLM 根据 description 决定何时调用。

## Skill 系统

Skill 是预定义的 Agent 角色配置，可让同一个 Agent 切换不同"人格"。

内置技能：
- `general` - 通用助手（默认）
- `coder` - 代码助手
- `translator` - 翻译官
- `analyst` - 数据分析师
- `storyteller` - 故事大王

交互命令：
- `skills` - 列出所有可用技能
- `skill <名称>` - 切换到指定技能

添加新技能：在 `agent/skill.go` 的 `RegisterBuiltinSkills()` 中注册。每个 Skill 包含 Name、Description、SystemPrompt 和可选的 Tools 列表。

## 代码风格

- 注释和文档为中文（教学项目）
- 标准 Go 错误处理：`fmt.Errorf` + `%w` 包装
- 全程使用 OpenAI API 兼容的请求/响应格式
