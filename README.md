# 🤖 AI Agent 教学 Demo (Go)

一个用 Go 实现的教学级 AI Agent，演示 **ReAct (Reasoning + Acting)** 模式的核心原理。

## 架构图

```
┌──────────────────────────────────────────────────────┐
│                    用户 (终端)                        │
│                 "帮我算 sqrt(144)"                   │
└───────────────────────┬──────────────────────────────┘
                        │
                        ▼
┌──────────────────────────────────────────────────────┐
│              Agent (ReAct 核心循环)                    │
│                                                      │
│  ┌─────────────────────────────────────────────┐     │
│  │  用户消息 → [system, user] → LLM            │     │
│  │                     │                        │     │
│  │            ┌────────┴────────┐               │     │
│  │            ▼                 ▼               │     │
│  │      有 tool_calls      没有 tool_calls     │     │
│  │            │                 │               │     │
│  │            ▼                 ▼               │     │
│  │     执行工具 → 结果      直接返回文本        │     │
│  │     → 加入历史                       ↓       │     │
│  │     → 继续循环                    最终答案   │     │
│  └─────────────────────────────────────────────┘     │
└──────────┬──────────────────────────┬────────────────┘
           │                          │
           ▼                          ▼
┌──────────────────┐    ┌──────────────────────────┐
│  LLM 客户端       │    │  工具系统                  │
│                  │    │                          │
│  • OpenAI API    │    │  • calculator (计算器)    │
│  • Ollama        │    │  • current_time (时间)    │
│  • Mock 模式     │    │  • search (搜索)          │
│                  │    │  • text_transform (文本)  │
└──────────────────┘    └──────────────────────────┘
```

## 快速开始

### Mock 模式（无需 API Key）

```bash
go run .
```

### 真实 LLM 模式

```bash
# OpenAI
go run . -api-key sk-xxx -model gpt-4o

# Ollama (本地)
go run . -api-key dummy -base-url http://localhost:11434/v1 -model qwen2

# 其他兼容 API
go run . -api-key xxx -base-url https://api.deepseek.com/v1 -model deepseek-chat
```

## 核心概念

### 1. 什么是 AI Agent？

```
AI Agent = LLM (大脑) + Tools (工具) + Loop (循环)
```

- **LLM**: 理解语言、推理、决策
- **Tools**: 与外部世界交互（计算、搜索、API调用...）
- **Loop**: 反复思考→行动→观察，直到完成任务

### 2. ReAct 模式

```
用户: "北京现在几点？"

THINK:  用户问时间，我需要获取当前时间
  ↓
ACT:    调用 current_time() → 返回 "14:30:25"
  ↓
OBSERVE: 现在是 14:30
  ↓
THINK:  信息够了，可以直接回答
  ↓
回复:   "现在是 14:30:25"
```

### 3. Function Calling

LLM 不直接执行工具，而是返回一个结构化的调用请求：

```json
{
  "tool_calls": [{
    "function": {
      "name": "calculator",
      "arguments": "{\"expression\": \"sqrt(144)\"}"
    }
  }]
}
```

Agent 框架负责执行，然后把结果喂回给 LLM。

## 项目结构

```
ai-agent-demo/
├── main.go              # 入口：交互式聊天界面
├── agent/
│   ├── types.go         # 类型定义：Message, ToolCall, Config
│   ├── tools.go         # 工具系统：注册表 + 内置工具
│   ├── llm.go           # LLM 客户端：OpenAI API / Mock
│   └── agent.go         # Agent 核心：ReAct 循环
└── README.md            # 本文件
```

## 交互命令

| 命令 | 说明 |
|------|------|
| `<任意文本>` | 向 Agent 提问 |
| `help` | 显示帮助 |
| `tools` | 列出可用工具 |
| `history` | 查看对话历史 |
| `reset` | 重置对话 |
| `quit` | 退出 |

## 内置工具

| 工具 | 说明 | 示例 |
|------|------|------|
| `calculator` | 数学计算 | `sqrt(144)`, `2 + 3 * 4` |
| `current_time` | 获取时间 | `Asia/Shanghai` |
| `search` | 模拟搜索 | `AI Agent` |
| `text_transform` | 文本处理 | 转大写/小写/反转/长度 |

## 扩展指南

### 添加新工具

在 `agent/tools.go` 的 `RegisterBuiltinTools` 中添加：

```go
registry.Register(Tool{
    Definition: ToolDefinition{
        Type: "function",
        Function: FunctionSchema{
            Name:        "my_tool",
            Description: "工具描述（LLM 靠这个决定用不用）",
            Parameters: json.RawMessage(`{
                "type": "object",
                "properties": {
                    "param1": {"type": "string", "description": "参数说明"}
                },
                "required": ["param1"]
            }`),
        },
    },
    Execute: func(args json.RawMessage) (string, error) {
        var params struct {
            Param1 string `json:"param1"`
        }
        json.Unmarshal(args, &params)
        return fmt.Sprintf("结果: %s", params.Param1), nil
    },
})
```

## 零依赖

本项目只使用 Go 标准库，无任何第三方依赖。
