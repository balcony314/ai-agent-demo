package agent

// ═══════════════════════════════════════════════════════════════
// tools.go — 工具注册表与内置工具
// ═══════════════════════════════════════════════════════════════
//
// 【教学要点】什么是工具（Tool）？
//
// 工具是 Agent 与外部世界交互的接口。
// LLM 本身只能生成文本，但通过工具它可以：
//   - 执行计算（calculator）
//   - 获取时间（current_time）
//   - 搜索信息（search）→ 这就是 RAG 的基础
//   - 操作文件、数据库、API...
//
// 工具系统的工作流程：
//   1. 开发者定义工具的 schema（名字、描述、参数）
//   2. Schema 发送给 LLM，LLM 决定什么时候用哪个工具
//   3. LLM 返回工具调用请求（name + arguments）
//   4. Agent 执行工具，拿到结果
//   5. 结果放回对话，LLM 继续推理
//
// 这就是 Function Calling 的核心机制！

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"
)

// ─── 工具注册表 ─────────────────────────────────────────────────

// ToolRegistry 管理所有可用工具
type ToolRegistry struct {
	tools map[string]Tool
}

// NewToolRegistry 创建一个新的工具注册表
func NewToolRegistry() *ToolRegistry {
	return &ToolRegistry{
		tools: make(map[string]Tool),
	}
}

// Register 注册一个工具
func (r *ToolRegistry) Register(tool Tool) {
	r.tools[tool.Definition.Function.Name] = tool
}

// Get 根据名称获取工具
func (r *ToolRegistry) Get(name string) (Tool, bool) {
	tool, ok := r.tools[name]
	return tool, ok
}

// Definitions 获取所有工具的 schema（发给 LLM 用）
func (r *ToolRegistry) Definitions() []ToolDefinition {
	defs := make([]ToolDefinition, 0, len(r.tools))
	for _, tool := range r.tools {
		defs = append(defs, tool.Definition)
	}
	return defs
}

// Names 获取所有工具名（日志用）
func (r *ToolRegistry) Names() []string {
	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	return names
}

// ─── 内置工具实现 ──────────────────────────────────────────────

// RegisterBuiltinTools 注册所有内置工具
func RegisterBuiltinTools(registry *ToolRegistry) {

	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	// 工具 1: 计算器
	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	registry.Register(Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "calculator",
				Description: "执行数学计算。支持加减乘除、幂运算、三角函数等。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"expression": {
							"type": "string",
							"description": "数学表达式，如 '2 + 3 * 4' 或 'sqrt(144)'"
						}
					},
					"required": ["expression"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Expression string `json:"expression"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}
			result, err := evaluateExpression(params.Expression)
			if err != nil {
				return "", fmt.Errorf("计算错误: %w", err)
			}
			return fmt.Sprintf("计算结果: %s = %g", params.Expression, result), nil
		},
	})

	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	// 工具 2: 获取当前时间
	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	registry.Register(Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "current_time",
				Description: "获取当前的日期和时间",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"timezone": {
							"type": "string",
							"description": "时区，如 'Asia/Shanghai'、'America/New_York'，默认本地时区"
						}
					},
					"required": []
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Timezone string `json:"timezone"`
			}
			json.Unmarshal(args, &params)

			loc := time.Now().Location()
			if params.Timezone != "" {
				var err error
				loc, err = time.LoadLocation(params.Timezone)
				if err != nil {
					return "", fmt.Errorf("无效的时区: %s", params.Timezone)
				}
			}
			now := time.Now().In(loc)
			return fmt.Sprintf("当前时间: %s (时区: %s)", now.Format("2006-01-02 15:04:05"), loc), nil
		},
	})

	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	// 工具 3: 模拟搜索（教学用）
	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	// 在真实场景中，这里会调用 Google/Bing API
	// 教学 demo 用一个小型知识库模拟
	registry.Register(Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "search",
				Description: "搜索信息。用于查找实时数据、百科知识等。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "搜索关键词"
						}
					},
					"required": ["query"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}
			return mockSearch(params.Query), nil
		},
	})

	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	// 工具 4: 字符串处理
	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	registry.Register(Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "text_transform",
				Description: "对文本进行转换：转大写、转小写、反转、计算长度",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"text": {
							"type": "string",
							"description": "要处理的文本"
						},
						"operation": {
							"type": "string",
							"enum": ["upper", "lower", "reverse", "length"],
							"description": "操作类型"
						}
					},
					"required": ["text", "operation"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Text      string `json:"text"`
				Operation string `json:"operation"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}
			switch params.Operation {
			case "upper":
				return fmt.Sprintf("大写: %s", strings.ToUpper(params.Text)), nil
			case "lower":
				return fmt.Sprintf("小写: %s", strings.ToLower(params.Text)), nil
			case "reverse":
				runes := []rune(params.Text)
				for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
					runes[i], runes[j] = runes[j], runes[i]
				}
				return fmt.Sprintf("反转: %s", string(runes)), nil
			case "length":
				return fmt.Sprintf("长度: %d 个字符", len([]rune(params.Text))), nil
			default:
				return "", fmt.Errorf("未知操作: %s", params.Operation)
			}
		},
	})

	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	// 工具 5-12: 文件操作工具集
	// ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
	RegisterFileTools(registry)
}

// ─── 简易数学表达式求值器 ──────────────────────────────────────
// 教学用，只支持基本运算

// evaluateExpression 是一个教学用的简易数学表达式求值器
// 支持：四则运算（+, -, *, /）和 sqrt() 函数
// 注意：这是为了演示工具实现而写的简化版本，生产环境应使用成熟的表达式解析库
func evaluateExpression(expr string) (float64, error) {
	expr = strings.TrimSpace(expr)

	// 优先处理 sqrt() 函数调用，提取括号内的表达式递归求值
	if strings.HasPrefix(expr, "sqrt(") && strings.HasSuffix(expr, ")") {
		inner := expr[5 : len(expr)-1]
		val, err := evaluateSimple(inner)
		if err != nil {
			return 0, err
		}
		if val < 0 {
			return 0, fmt.Errorf("不能对负数开平方")
		}
		return math.Sqrt(val), nil
	}

	return evaluateSimple(expr)
}

// evaluateSimple 解析 "a op b" 格式的简单四则运算
// 用 LastIndex 找到最后一个运算符，这样能正确处理 "1 + 2 * 3"（先算乘法）
func evaluateSimple(expr string) (float64, error) {
	var a, b float64
	var op string

	// 遍历运算符，找到最后一个作为分割点
	for _, o := range []string{"+", "-", "*", "/"} {
		idx := strings.LastIndex(expr, o)
		if idx > 0 && idx < len(expr)-1 {
			left := strings.TrimSpace(expr[:idx])
			right := strings.TrimSpace(expr[idx+len(o):])
			var err error
			a, err = parseFloat(left)
			if err != nil {
				return 0, fmt.Errorf("无效的数字: %s", left)
			}
			b, err = parseFloat(right)
			if err != nil {
				return 0, fmt.Errorf("无效的数字: %s", right)
			}
			op = o
			break
		}
	}

	if op == "" {
		return parseFloat(expr)
	}

	switch op {
	case "+":
		return a + b, nil
	case "-":
		return a - b, nil
	case "*":
		return a * b, nil
	case "/":
		if b == 0 {
			return 0, fmt.Errorf("除以零")
		}
		return a / b, nil
	}
	return 0, fmt.Errorf("不支持的运算符: %s", op)
}

func parseFloat(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}

// ─── 模拟搜索引擎 ──────────────────────────────────────────────
// 真实场景中这里会调用 Google/Bing API 或向量数据库
// 教学 demo 用关键词匹配模拟搜索结果

func mockSearch(query string) string {
	// 教学用的小型知识库：键是关键词，值是模拟的搜索结果
	knowledge := map[string]string{
		"golang":    "Go 是 Google 开发的编程语言，特点是简洁、高效、并发友好。最新版本 Go 1.23 引入了迭代器等新特性。",
		"python":    "Python 是一种解释型、面向对象的高级编程语言。以简洁易读著称，广泛用于 AI/ML、Web 开发、数据科学。",
		"agent":     "AI Agent（智能体）是能够感知环境、做出决策并采取行动来实现目标的系统。核心组件：LLM（大脑）+ Tools（手脚）+ Memory（记忆）。",
		"react":     "ReAct (Reasoning + Acting) 是一种让 LLM 交替进行推理和行动的范式。思考→行动→观察→再思考，形成循环。",
		"mcp":       "Model Context Protocol (MCP) 是 Anthropic 提出的开放协议，用于标准化 AI 模型与外部工具/数据源的连接方式。",
		"llm":       "大型语言模型 (LLM) 是基于 Transformer 架构、在海量文本上训练的深度学习模型。代表：GPT-4, Claude, LLaMA。",
		"transformer": "Transformer 是 Google 在 2017 年提出的神经网络架构，通过自注意力机制处理序列数据，是现代 LLM 的基础。",
	}

	query = strings.ToLower(query)
	for key, value := range knowledge {
		if strings.Contains(query, key) {
			return fmt.Sprintf("搜索结果 '%s':\n%s", query, value)
		}
	}
	return fmt.Sprintf("搜索 '%s': 没有找到精确匹配的结果。建议尝试更具体的关键词。", query)
}
