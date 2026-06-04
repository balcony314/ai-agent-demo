# CLAUDE.md

本文件为 Claude Code (claude.ai/code) 在本仓库中工作提供指引。

## 项目概述

用 Go 实现的教学级 AI Agent，演示 **ReAct (Reasoning + Acting)** 模式。通过 OpenAI 兼容的 function calling 实现带工具调用（计算器、时间、搜索、文本处理）的对话式 Agent。零第三方依赖，仅用标准库。

## 常用命令

```bash
go run .                                          # Mock 模式运行（无需 API key）
go run . -api-key sk-xxx -model gpt-4o            # 接 OpenAI
go run . -api-key dummy -base-url http://localhost:11434/v1 -model qwen2  # 接 Ollama
go build -o ai-agent .                            # 编译
go test ./...                                     # 测试（目前无测试文件）
go vet ./...                                      # 静态检查
```

CLI 参数：`-api-key`、`-base-url`、`-model`（默认 `gpt-4o`）、`-mock`

## 架构

核心是 ReAct 循环：LLM 生成文本或工具调用 → Agent 执行工具 → 把结果喂回 LLM → 循环（最多 10 轮），直到 LLM 输出最终文本回答。

```
main.go           → 入口：CLI 参数 + 交互式 REPL
agent/
  types.go        → 核心类型：Message, ToolCall, ToolDefinition, Config, LLMClient 接口
  tools.go        → ToolRegistry（map 实现）+ RegisterBuiltinTools() 注册 4 个内置工具
  llm.go          → OpenAIClient（真实 /v1/chat/completions）+ MockClient（演示用）
  agent.go        → Agent.Run() —— ReAct 编排循环
```

关键抽象：
- **`LLMClient` 接口**（`Chat` 方法）—— 可在 `OpenAIClient` 和 `MockClient` 之间切换
- **`ToolRegistry`** —— `map[string]Tool`，提供 `Register`/`Get`/`Definitions`；每个 `Tool` 由 JSON Schema 的 `ToolDefinition` + `Execute` 函数组成
- **`Message`** —— role + content + 可选的 tool_calls/tool_call_id，对齐 OpenAI chat 格式

## 添加新工具

在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中注册。提供带 JSON Schema 参数的 `ToolDefinition` 和一个 `Execute` 函数。LLM 根据 description 决定何时调用。

## 代码风格

- 注释和文档为中文（教学项目）
- 标准 Go 错误处理：`fmt.Errorf` + `%w` 包装
- 全程使用 OpenAI API 兼容的请求/响应格式
