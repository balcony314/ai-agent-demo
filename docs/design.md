# 源码设计思路

## 项目定位

一个教学级 AI Agent 框架，用 Go 语言实现 **ReAct (Reasoning + Acting)** 模式。目标是让开发者通过阅读源码理解 AI Agent 的核心原理，而非构建生产级系统。

## 整体架构

```
┌──────────────────────────────────────────────────────────────┐
│                        用户 (终端)                           │
│                     "帮我算一下 sqrt(144)"                    │
└──────────────────────────┬───────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                    cmd/ (CLI 层)                              │
│                                                              │
│  root.go ──── 交互式 REPL（默认）                            │
│  chat.go ──── 单次提问（子命令）                              │
│  version.go ─ 版本信息                                       │
│                                                              │
│  职责：解析参数、创建 LLM 客户端、驱动 Agent                  │
└──────────────────────────┬───────────────────────────────────┘
                           │
                           ▼
┌──────────────────────────────────────────────────────────────┐
│                    agent/ (核心层)                            │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  Agent (agent.go)                                   │     │
│  │                                                     │     │
│  │  Run(userInput) → ReAct 循环                        │     │
│  │                                                     │     │
│  │  ┌───────────────────────────────────────────┐      │     │
│  │  │  1. 用户消息加入 history                  │      │     │
│  │  │  2. 发送 history + tools → LLM            │      │     │
│  │  │  3. LLM 返回 tool_calls?                  │      │     │
│  │  │     ├─ 是 → 执行工具 → 结果加入 history   │      │     │
│  │  │     │       → 回到第 2 步                 │      │     │
│  │  │     └─ 否 → 返回 content 作为最终回答     │      │     │
│  │  └───────────────────────────────────────────┘      │     │
│  └─────────────────────────────────────────────────────┘     │
│              │                    │                          │
│              ▼                    ▼                          │
│  ┌──────────────────┐  ┌──────────────────────────┐         │
│  │  LLMClient       │  │  ToolRegistry            │         │
│  │  (llm.go)        │  │  (tools.go)              │         │
│  │                  │  │                          │         │
│  │  OpenAIClient    │  │  calculator              │         │
│  │  MockClient      │  │  current_time            │         │
│  └──────────────────┘  │  search                  │         │
│                        │  text_transform          │         │
│                        └──────────────────────────┘         │
│                                                              │
│  ┌─────────────────────────────────────────────────────┐     │
│  │  SkillRegistry (skill.go)                           │     │
│  │                                                     │     │
│  │  general / coder / translator / analyst / storyteller│    │
│  │  每个 Skill = SystemPrompt + 可用工具列表           │     │
│  └─────────────────────────────────────────────────────┘     │
│                                                              │
│  types.go ──── 核心类型：Message, ToolCall, Config 等        │
└──────────────────────────────────────────────────────────────┘
```

## 核心概念

### ReAct 循环

ReAct = **Re**asoning + **Act**ing，是让 LLM 从"只能说"变成"能做"的关键范式。

```
用户: "北京现在几点？10分钟前呢？"

┌─── ReAct Loop ───────────────────────────────────┐
│                                                   │
│  THINK: 用户问时间，需要先获取当前时间             │
│    → 决定调用 current_time 工具                   │
│                                                   │
│  ACT: 调用 current_time({timezone: "Asia/Shanghai"})│
│    → 返回 "2024-01-15 14:30:25"                   │
│                                                   │
│  OBSERVE: 当前是 14:30，10分钟前是 14:20          │
│                                                   │
│  THINK: 信息足够，生成最终回复                     │
│                                                   │
└───────────────────────────────────────────────────┘

Agent: "现在是 14:30，10分钟前是 14:20"
```

关键洞察：LLM 不直接输出答案，而是输出 **"我想要做什么"**（tool_calls），Agent 框架执行后把结果喂回去，LLM 再决定下一步。

### Function Calling

Function Calling 是 LLM 与工具交互的协议：

1. 开发者定义工具的 JSON Schema（名称、描述、参数）
2. Schema 随请求发给 LLM
3. LLM 返回 `tool_calls` 而非文本（表示"我要调用这个工具"）
4. Agent 执行工具，把结果作为 `RoleTool` 消息放回对话
5. LLM 看到工具结果后继续推理

这就是 `agent/types.go` 中 `ToolDefinition`、`ToolCall`、`Message` 这些类型的用途。

### Skill 系统

Skill 是 Agent 的"人格切换"机制。每个 Skill 包含：
- **SystemPrompt**: 告诉 LLM 它现在是什么角色
- **Tools**: 该角色可用的工具列表（空 = 全部）

切换 Skill 会重置对话历史，让 Agent 以全新身份开始。

## 模块详解

### types.go — 类型基础

定义了整个 Agent 的数据结构：

| 类型 | 用途 |
|------|------|
| `Message` | 对话消息，对齐 OpenAI chat 格式 |
| `Role` | 消息角色：system / user / assistant / tool |
| `ToolCall` | LLM 请求调用的工具（ID + 函数名 + 参数） |
| `ToolDefinition` | 工具的 JSON Schema 描述（发给 LLM） |
| `Tool` | 完整工具 = Definition + Execute 函数 |
| `Config` | Agent 配置：SystemPrompt + MaxTurns |

**设计决策**：所有类型直接对齐 OpenAI API 格式，避免额外转换层。

### llm.go — LLM 客户端

通过 `LLMClient` 接口抽象 LLM 调用：

```go
type LLMClient interface {
    Chat(messages []Message, tools []ToolDefinition) (*Message, error)
}
```

两种实现：
- **OpenAIClient**: 调用 `/v1/chat/completions`，兼容 OpenAI/Ollama/vLLM
- **MockClient**: 无需 API Key，按轮次模拟工具调用，用于教学演示

**设计决策**：用接口而非具体实现，让 Agent 与 LLM 解耦。替换后端只需实现 `Chat` 方法。

### tools.go — 工具系统

采用**注册表模式**管理工具：

```go
type ToolRegistry struct {
    tools map[string]Tool  // 名称 → 工具
}
```

内置 4 个工具：

| 工具 | 功能 | 参数 |
|------|------|------|
| `calculator` | 数学计算 | `expression: string` |
| `current_time` | 获取时间 | `timezone?: string` |
| `search` | 模拟搜索 | `query: string` |
| `text_transform` | 文本转换 | `text: string, operation: enum` |

**设计决策**：注册表模式让工具可动态添加，LLM 通过 `Definitions()` 获取所有工具的 schema。

### skill.go — 技能系统

同样采用注册表模式：

```go
type SkillRegistry struct {
    skills map[string]Skill
}
```

内置 5 个技能：general、coder、translator、analyst、storyteller。

**设计决策**：Skill 只是配置（SystemPrompt + 工具列表），不含执行逻辑，保持简单。

### agent.go — 编排核心

Agent 是整个系统的中枢，组合了：
- `LLMClient`（大脑）
- `ToolRegistry`（手脚）
- `SkillRegistry`（人格）
- `history`（记忆）

`Run()` 方法实现 ReAct 循环，是理解整个系统的关键入口。

## 数据流

一次完整的 Agent 调用：

```
用户输入 "帮我算 sqrt(144)"
    │
    ▼
Agent.Run() 加入 history
    │
    ▼
LLM.Chat(history, tools)
    │
    ├─ LLM 返回 tool_calls: [{name: "calculator", args: {"expression": "sqrt(144)"}}]
    │
    ▼
Agent.executeTool("calculator", args)
    │
    ▼
Tool.Execute(args) → "计算结果: sqrt(144) = 12"
    │
    ▼
结果作为 RoleTool 消息加入 history
    │
    ▼
LLM.Chat(history, tools)  ← 第二轮
    │
    ├─ LLM 返回 content: "sqrt(144) 的结果是 12"
    │
    ▼
返回最终回答
```

## 设计决策

### 1. 接口抽象 LLM

`LLMClient` 接口让 Agent 不依赖具体 LLM 实现。好处：
- 测试时可用 MockClient
- 切换后端（OpenAI → Ollama）零改动
- 未来可加 streaming、retry 等实现

### 2. 注册表模式

`ToolRegistry` 和 `SkillRegistry` 都用 map + Register/Get 模式。好处：
- 运行时动态注册
- 按名称查找 O(1)
- 便于扩展

### 3. 对齐 OpenAI 格式

所有消息类型直接兼容 OpenAI chat API。好处：
- 无需转换层
- 容易对接现有生态
- 教学时可直接对照 OpenAI 文档

### 4. Skill 作为配置而非继承

Skill 不是 Agent 的子类，只是一组配置。好处：
- 切换成本低（改 SystemPrompt + 过滤工具）
- 一个 Agent 可以有任意多 Skill
- 避免类继承的复杂性

## 扩展指南

### 添加新工具

在 `agent/tools.go` 的 `RegisterBuiltinTools()` 中添加：

```go
registry.Register(Tool{
    Definition: ToolDefinition{
        Type: "function",
        Function: FunctionSchema{
            Name:        "my_tool",
            Description: "工具描述（LLM 根据这个决定何时调用）",
            Parameters:  json.RawMessage(`{"type": "object", "properties": {...}}`),
        },
    },
    Execute: func(args json.RawMessage) (string, error) {
        // 解析参数、执行逻辑、返回结果
        return "result", nil
    },
})
```

### 添加新技能

在 `agent/skill.go` 的 `RegisterBuiltinSkills()` 中添加：

```go
registry.Register(Skill{
    Name:         "my_skill",
    Description:  "技能描述",
    SystemPrompt: "你是一个...",
    Tools:        []string{"calculator"}, // 空 = 全部工具
})
```

### 替换 LLM 后端

实现 `LLMClient` 接口即可：

```go
type MyClient struct{}

func (c *MyClient) Chat(messages []Message, tools []ToolDefinition) (*Message, error) {
    // 调用你的 LLM API
    return &Message{Role: RoleAssistant, Content: "reply"}, nil
}
```
