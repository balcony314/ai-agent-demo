package main

// ═══════════════════════════════════════════════════════════════
// main.go — AI Agent 教学 Demo 入口
// ═══════════════════════════════════════════════════════════════
//
// 【整体架构图】
//
//   ┌──────────────────────────────────────────────────────┐
//   │                    用户 (终端)                       │
//   │                   "你好，现在几点？"                  │
//   └───────────────────────┬──────────────────────────────┘
//                           │
//                           ▼
//   ┌──────────────────────────────────────────────────────┐
//   │                  main.go (交互层)                     │
//   │              读取输入 → 调用 Agent → 打印结果         │
//   └───────────────────────┬──────────────────────────────┘
//                           │
//                           ▼
//   ┌──────────────────────────────────────────────────────┐
//   │              agent.go (ReAct 核心)                    │
//   │                                                      │
//   │  ┌─────────────────────────────────────────────┐     │
//   │  │  用户消息 → [system, user] → LLM            │     │
//   │  │                     │                        │     │
//   │  │            ┌────────┴────────┐               │     │
//   │  │            ▼                 ▼               │     │
//   │  │      有 tool_calls      没有 tool_calls     │     │
//   │  │            │                 │               │     │
//   │  │            ▼                 ▼               │     │
//   │  │     执行工具 → 结果      直接返回文本        │     │
//   │  │     → 加入历史                       ↓       │     │
//   │  │     → 继续循环                    最终答案   │     │
//   │  └─────────────────────────────────────────────┘     │
//   └──────────┬──────────────────────────┬────────────────┘
//              │                          │
//              ▼                          ▼
//   ┌──────────────────┐    ┌──────────────────────────┐
//   │  llm.go (LLM)    │    │  tools.go (工具系统)      │
//   │                  │    │                          │
//   │  OpenAI API   ───┼───→│  calculator             │
//   │  或 Mock 模式     │    │  current_time           │
//   │                  │    │  search                  │
//   └──────────────────┘    │  text_transform          │
//                           └──────────────────────────┘
//
// 使用方式:
//   go run .                    # Mock 模式（无需 API Key）
//   go run . -api-key sk-xxx    # 真实 LLM 模式
//   go run . -base-url http://localhost:11434/v1 -model qwen2  # Ollama

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"strings"

	"ai-agent-demo/agent"
)

func main() {
	// ─── 命令行参数 ──────────────────────────────────────────────
	apiKey := flag.String("api-key", "", "LLM API Key（为空则使用 Mock 模式）")
	baseURL := flag.String("base-url", "", "API 地址（默认 OpenAI，可设为 Ollama 等）")
	model := flag.String("model", "gpt-4o", "模型名称")
	mockMode := flag.Bool("mock", false, "强制使用 Mock 模式（无需 API Key）")
	flag.Parse()

	// ─── 打印启动信息 ────────────────────────────────────────────
	printBanner()

	// ─── 创建 LLM 客户端 ─────────────────────────────────────────
	// LLMClient 是一个接口，有两种实现：
	//   - MockClient: 无需 API Key，用预设逻辑模拟 LLM 行为，适合教学演示
	//   - OpenAIClient: 调用真实的 OpenAI 兼容 API（支持 OpenAI/Ollama/vLLM 等）
	var llm agent.LLMClient

	if *mockMode || *apiKey == "" {
		fmt.Println("🤖 LLM 模式: Mock（教学演示）")
		fmt.Println("   提示: 设置 -api-key 参数可连接真实 LLM API")
		fmt.Println("   支持: OpenAI, Ollama, vLLM, LiteLLM 等兼容 API")
		fmt.Println()
		llm = agent.NewMockClient()
	} else {
		fmt.Printf("🤖 LLM 模式: 真实 API\n")
		fmt.Printf("   模型: %s\n", *model)
		if *baseURL != "" {
			fmt.Printf("   地址: %s\n", *baseURL)
		}
		fmt.Println()
		llm = agent.NewOpenAIClient(*apiKey, *baseURL, *model)
	}

	// ─── 创建 Agent ──────────────────────────────────────────────
	// Agent = LLM（大脑） + Tools（工具） + ReAct Loop（推理循环）
	// config 中的 SystemPrompt 告诉 LLM 它是一个 AI 助手，可以使用工具
	config := agent.DefaultConfig()
	a := agent.NewAgent(llm, config)

	fmt.Printf("🛠️  可用工具: %s\n", strings.Join(a.ListTools(), ", "))
	fmt.Printf("✨ 当前技能: %s\n", a.CurrentSkill())
	fmt.Println()
	fmt.Println("💡 输入问题开始对话，输入 'quit' 退出，'reset' 重置对话")
	fmt.Println("   输入 'skills' 查看可用技能，'skill <名称>' 切换技能")
	fmt.Println(strings.Repeat("═", 55))

	// ─── 交互式聊天循环 ──────────────────────────────────────────
	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\n🧑 你> ")
		if !scanner.Scan() {
			break
		}
		input := strings.TrimSpace(scanner.Text())

		if input == "" {
			continue
		}

		// 特殊命令
		cmd := strings.ToLower(input)
		switch {
		case cmd == "quit" || cmd == "exit" || cmd == "q":
			fmt.Println("\n👋 再见！")
			return
		case cmd == "reset" || cmd == "clear":
			a.ClearHistory()
			fmt.Println("🔄 对话已重置")
			continue
		case cmd == "history":
			printHistory(a.GetHistory())
			continue
		case cmd == "tools":
			fmt.Printf("🛠️  可用工具: %s\n", strings.Join(a.ListTools(), ", "))
			continue
		case cmd == "skills":
			fmt.Print(a.FormatSkillList())
			continue
		case cmd == "help":
			printHelp()
			continue
		case strings.HasPrefix(cmd, "skill "):
			skillName := strings.TrimSpace(strings.TrimPrefix(input, "skill "))
			if err := a.SwitchSkill(skillName); err != nil {
				fmt.Printf("❌ %v\n", err)
			}
			continue
		}

		// 运行 Agent 的 ReAct 循环：
		//   1. 把用户消息发给 LLM
		//   2. 如果 LLM 返回 tool_calls → 执行工具 → 结果喂回 LLM → 继续循环
		//   3. 如果 LLM 返回文本 → 作为最终答案返回
		reply, err := a.Run(input)
		if err != nil {
			fmt.Printf("\n❌ 错误: %v\n", err)
			continue
		}

		fmt.Printf("\n🤖 助手> %s\n", reply)
	}

	fmt.Println("\n👋 再见！")
}

// ─── 界面辅助函数 ──────────────────────────────────────────────

func printBanner() {
	fmt.Print(`
╔══════════════════════════════════════════════════════╗
║           🤖 AI Agent 教学 Demo (Go)                ║
║                                                      ║
║   核心概念: ReAct Loop (Reasoning + Acting)          ║
║   架构: LLM + Tools + 智能循环                       ║
║   传输: stdio 交互式                                 ║
╚══════════════════════════════════════════════════════╝
`)
}

// printHelp 显示帮助信息
// 列出所有可用的交互命令和示例问题
func printHelp() {
	fmt.Print(`
┌─────────────────────────────────────────────────────┐
│  📖 可用命令                                         │
├─────────────────────────────────────────────────────┤
│  <任意文本>    向 Agent 提问                         │
│  help          显示此帮助信息                         │
│  tools         列出所有可用工具                       │
│  skills        列出所有可用技能                       │
│  skill <名称>  切换到指定技能                         │
│  history       查看完整对话历史                       │
│  reset         重置对话（清空历史）                    │
│  quit          退出程序                              │
└─────────────────────────────────────────────────────┘

💡 示例问题:
  • 你好
  • 帮我算一下 sqrt(144) * 3 + 7
  • 现在几点了？
  • 搜索一下什么是 AI Agent
  • 把 "hello world" 转大写

🎭 技能说明:
  • general    - 通用助手（默认）
  • coder      - 代码助手
  • translator - 翻译官
  • analyst    - 数据分析师
  • storyteller - 故事大王
`)
}

// printHistory 打印对话历史
// 不同角色的消息用不同前缀标识：
//   - System: 系统提示词（用户一般看不到）
//   - User: 用户输入
//   - Assistant: LLM 回复（可能包含 tool_calls）
//   - Tool: 工具执行结果
func printHistory(history []agent.Message) {
	fmt.Println("\n📜 对话历史:")
	fmt.Println(strings.Repeat("─", 50))
	for i, msg := range history {
		var prefix string
		switch msg.Role {
		case agent.RoleSystem:
			prefix = "⚙️  [System]"
		case agent.RoleUser:
			prefix = "🧑 [User]"
		case agent.RoleAssistant:
			prefix = "🤖 [Assistant]"
		case agent.RoleTool:
			prefix = "🔧 [Tool]"
		default:
			prefix = fmt.Sprintf("[%s]", msg.Role)
		}
		content := msg.Content
		if content == "" && len(msg.ToolCalls) > 0 {
			names := make([]string, len(msg.ToolCalls))
			for j, tc := range msg.ToolCalls {
				names[j] = tc.Function.Name
			}
			content = fmt.Sprintf("(调用工具: %s)", strings.Join(names, ", "))
		}
		fmt.Printf("  %d. %s %s\n", i, prefix, agent.TruncStr(content, 100))
	}
	fmt.Println(strings.Repeat("─", 50))
}
