package cmd

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"testing"

	"ai-agent-demo/agent"
)

func TestInitLLMClient_MockMode(t *testing.T) {
	// 强制 Mock 模式
	mock = true
	apiKey = ""

	client := initLLMClient()
	if client == nil {
		t.Fatal("initLLMClient 返回 nil")
	}

	// 验证返回的是 MockClient
	if _, ok := client.(*agent.MockClient); !ok {
		t.Error("预期返回 MockClient")
	}
}

func TestInitLLMClient_NoAPIKey(t *testing.T) {
	// 没有 API Key 时应该使用 Mock 模式
	mock = false
	apiKey = ""

	client := initLLMClient()
	if client == nil {
		t.Fatal("initLLMClient 返回 nil")
	}

	if _, ok := client.(*agent.MockClient); !ok {
		t.Error("没有 API Key 时应返回 MockClient")
	}
}

func TestInitLLMClient_WithAPIKey(t *testing.T) {
	// 有 API Key 时应该创建 OpenAI 客户端
	mock = false
	apiKey = "sk-test-key"
	baseURL = ""
	model = "gpt-4o"

	client := initLLMClient()
	if client == nil {
		t.Fatal("initLLMClient 返回 nil")
	}

	if _, ok := client.(*agent.OpenAIClient); !ok {
		t.Error("有 API Key 时应返回 OpenAIClient")
	}
}

func TestInitLLMClient_WithBaseURL(t *testing.T) {
	// 测试自定义 base URL
	mock = false
	apiKey = "sk-test-key"
	baseURL = "http://localhost:11434/v1"
	model = "qwen2"

	client := initLLMClient()
	if client == nil {
		t.Fatal("initLLMClient 返回 nil")
	}

	if _, ok := client.(*agent.OpenAIClient); !ok {
		t.Error("应返回 OpenAIClient")
	}
}

func TestPrintBanner(t *testing.T) {
	// printBanner 只是打印，确保不 panic
	printBanner()
}

func TestPrintHelp(t *testing.T) {
	// printHelp 只是打印，确保不 panic
	printHelp()
}

func TestPrintHistory(t *testing.T) {
	// 测试空历史
	printHistory([]agent.Message{})

	// 测试有消息的历史
	history := []agent.Message{
		{Role: agent.RoleUser, Content: "你好"},
		{Role: agent.RoleAssistant, Content: "你好！有什么可以帮你的吗？"},
		{Role: agent.RoleTool, Content: "工具结果"},
	}
	printHistory(history)
}

func TestPrintHistory_WithToolCalls(t *testing.T) {
	// 测试包含工具调用的历史
	history := []agent.Message{
		{
			Role:    agent.RoleAssistant,
			Content: "",
			ToolCalls: []agent.ToolCall{
				{ID: "call_1", Function: agent.FunctionCall{Name: "calculator", Arguments: `{"expression":"1+1"}`}},
			},
		},
	}
	printHistory(history)
}

func TestPrintHistory_UnknownRole(t *testing.T) {
	// 测试未知 role 的 default 分支
	history := []agent.Message{
		{Role: agent.Role("custom"), Content: "自定义角色消息"},
	}
	printHistory(history)
}

// ─── checkAPIConnection 测试 ──────────────────────────────────

func TestCheckAPIConnection_MockClient(t *testing.T) {
	// MockClient 应该直接跳过检查
	client := agent.NewMockClient()
	checkAPIConnection(client) // 不应 panic 或退出
}

func TestCheckAPIConnection_OpenAIClientSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := agent.NewOpenAIClient("sk-test", server.URL, "gpt-4o")
	checkAPIConnection(client) // 不应退出
}

// checkAPIConnection 失败时调用 os.Exit(1)，需要用子进程测试
func TestCheckAPIConnection_Failure(t *testing.T) {
	if os.Getenv("TEST_CHECK_API_FAIL") == "1" {
		// 子进程：连接一个不存在的地址，触发 os.Exit(1)
		client := agent.NewOpenAIClient("sk-test", "http://127.0.0.1:1", "gpt-4o")
		checkAPIConnection(client)
		return
	}

	// 主进程：启动子进程并验证退出码
	cmd := exec.Command(os.Args[0], "-test.run=TestCheckAPIConnection_Failure")
	cmd.Env = append(os.Environ(), "TEST_CHECK_API_FAIL=1")
	err := cmd.Run()
	if err == nil {
		t.Fatal("子进程应以非零退出码退出")
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("预期 ExitError，实际: %v", err)
	}
	if exitErr.ExitCode() != 1 {
		t.Errorf("退出码 = %d, 期望 1", exitErr.ExitCode())
	}
}
