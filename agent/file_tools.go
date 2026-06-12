package agent

// ═══════════════════════════════════════════════════════════════
// file_tools.go — 文件操作工具集
// ═══════════════════════════════════════════════════════════════
//
// 提供 8 个文件操作工具，用于 AI Agent 进行文件系统操作。
// 所有操作都限制在当前工作目录内，防止路径遍历攻击。
//
// 工具列表：
//   - read_file: 读取文件内容
//   - write_file: 写入文件（创建/覆盖）
//   - edit_file: 编辑文件（替换/追加/插入/删除）
//   - list_dir: 列出目录内容
//   - file_info: 获取文件信息
//   - delete_file: 删除文件或目录
//   - search_files: 按名称模式搜索文件
//   - search_content: 按内容搜索（支持正则）

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// ─── 路径安全验证 ─────────────────────────────────────────────

// getWorkDir 获取当前工作目录（支持测试时模拟）
var getWorkDir = func() string {
	dir, _ := os.Getwd()
	return dir
}

// validatePath 验证并规范化路径，确保在工作目录内
func validatePath(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("路径不能为空")
	}

	if strings.Contains(path, "..") {
		return "", fmt.Errorf("路径不允许包含 '..': %s", path)
	}

	workDir := getWorkDir()
	var absPath string
	if filepath.IsAbs(path) {
		absPath = filepath.Clean(path)
	} else {
		absPath = filepath.Clean(filepath.Join(workDir, path))
	}

	if !strings.HasPrefix(absPath, workDir) {
		return "", fmt.Errorf("路径超出工作目录范围: %s (工作目录: %s)", path, workDir)
	}

	return absPath, nil
}

// isTextFile 检查文件是否为文本文件（通过扩展名判断）
func isTextFile(path string) bool {
	textExts := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".java": true, ".c": true, ".cpp": true, ".h": true,
		".rs": true, ".rb": true, ".php": true, ".swift": true,
		".kt": true, ".scala": true, ".sh": true, ".bash": true,
		".zsh": true, ".fish": true, ".ps1": true, ".bat": true,
		".cmd": true, ".sql": true, ".html": true, ".css": true,
		".scss": true, ".less": true, ".xml": true, ".json": true,
		".yaml": true, ".yml": true, ".toml": true, ".ini": true,
		".cfg": true, ".conf": true, ".md": true, ".txt": true,
		".rst": true, ".csv": true, ".log": true,
		".gitignore": true, ".dockerignore": true, ".editorconfig": true,
		".prettierrc": true, ".eslintrc": true, ".babelrc": true,
		".vue": true, ".jsx": true, ".tsx": true, ".svelte": true,
		".astro": true, ".graphql": true, ".proto": true,
		".makefile": true, ".dockerfile": true,
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == "" {
		base := strings.ToLower(filepath.Base(path))
		return base == "makefile" || base == "dockerfile" ||
			base == "license" || base == "readme" ||
			base == "changelog" || base == "authors" ||
			base == "contributing"
	}
	return textExts[ext]
}

// formatSize 格式化文件大小为人类可读格式
func formatSize(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %s", float64(size)/float64(div), []string{"KB", "MB", "GB", "TB", "PB", "EB"}[exp])
}

// ─── 工具注册 ─────────────────────────────────────────────────

// RegisterFileTools 注册所有文件操作工具
func RegisterFileTools(registry *ToolRegistry) {
	registry.Register(newReadFileTool())
	registry.Register(newWriteFileTool())
	registry.Register(newEditFileTool())
	registry.Register(newListDirTool())
	registry.Register(newFileInfoTool())
	registry.Register(newDeleteFileTool())
	registry.Register(newSearchFilesTool())
	registry.Register(newSearchContentTool())
}

// ─── 工具实现 ─────────────────────────────────────────────────

// newReadFileTool 创建读取文件工具
func newReadFileTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "read_file",
				Description: "读取文件内容。仅支持文本文件，最大 1MB。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "文件路径（相对于工作目录）"
						}
					},
					"required": ["path"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			absPath, err := validatePath(params.Path)
			if err != nil {
				return "", err
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("文件不存在: %s", params.Path)
				}
				return "", fmt.Errorf("获取文件信息失败: %w", err)
			}

			if info.IsDir() {
				return "", fmt.Errorf("路径是目录，不是文件: %s", params.Path)
			}

			const maxSize = 1024 * 1024
			if info.Size() > maxSize {
				return "", fmt.Errorf("文件过大 (%s)，最大支持 1MB", formatSize(info.Size()))
			}

			if !isTextFile(absPath) {
				return "", fmt.Errorf("不支持的文件类型，仅支持文本文件: %s", params.Path)
			}

			content, err := os.ReadFile(absPath)
			if err != nil {
				return "", fmt.Errorf("读取文件失败: %w", err)
			}

			lines := strings.Count(string(content), "\n") + 1
			return fmt.Sprintf("文件内容 (%s, %d 行):\n%s", params.Path, lines, string(content)), nil
		},
	}
}

// newWriteFileTool 创建写入文件工具
func newWriteFileTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "write_file",
				Description: "写入文件（创建或覆盖）。自动创建父目录。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "文件路径（相对于工作目录）"
						},
						"content": {
							"type": "string",
							"description": "要写入的内容"
						}
					},
					"required": ["path", "content"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			absPath, err := validatePath(params.Path)
			if err != nil {
				return "", err
			}

			if !isTextFile(absPath) {
				return "", fmt.Errorf("不支持的文件类型，仅支持文本文件: %s", params.Path)
			}

			if err := os.MkdirAll(filepath.Dir(absPath), 0755); err != nil {
				return "", fmt.Errorf("创建目录失败: %w", err)
			}

			if err := os.WriteFile(absPath, []byte(params.Content), 0644); err != nil {
				return "", fmt.Errorf("写入文件失败: %w", err)
			}

			return fmt.Sprintf("文件已写入: %s (%d 字节)", params.Path, len(params.Content)), nil
		},
	}
}

// newEditFileTool 创建编辑文件工具
func newEditFileTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "edit_file",
				Description: "编辑文件。支持 4 种操作：replace（查找替换）、append（追加）、insert（插入行）、delete（删除行）。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "文件路径（相对于工作目录）"
						},
						"action": {
							"type": "string",
							"enum": ["replace", "append", "insert", "delete"],
							"description": "编辑操作类型"
						},
						"find": {
							"type": "string",
							"description": "要查找的文本（action=replace 时必填）"
						},
						"replace_with": {
							"type": "string",
							"description": "替换后的文本（action=replace 时必填）"
						},
						"content": {
							"type": "string",
							"description": "要追加或插入的内容（action=append/insert 时必填）"
						},
						"line": {
							"type": "integer",
							"description": "行号，从 1 开始（action=insert/delete 时必填）"
						},
						"end_line": {
							"type": "integer",
							"description": "结束行号，包含该行（action=delete 时可选，默认等于 line）"
						}
					},
					"required": ["path", "action"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path        string `json:"path"`
				Action      string `json:"action"`
				Find        string `json:"find"`
				ReplaceWith string `json:"replace_with"`
				Content     string `json:"content"`
				Line        int    `json:"line"`
				EndLine     int    `json:"end_line"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			absPath, err := validatePath(params.Path)
			if err != nil {
				return "", err
			}

			// 读取文件内容
			content, err := os.ReadFile(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("文件不存在: %s", params.Path)
				}
				return "", fmt.Errorf("读取文件失败: %w", err)
			}

			lines := strings.Split(string(content), "\n")
			var result string

			switch params.Action {
			case "replace":
				if params.Find == "" {
					return "", fmt.Errorf("replace 操作需要 find 参数")
				}
				newContent := strings.ReplaceAll(string(content), params.Find, params.ReplaceWith)
				count := strings.Count(string(content), params.Find) - strings.Count(newContent, params.Find)
				if count < 0 {
					count = 0
				}
				lines = strings.Split(newContent, "\n")
				result = fmt.Sprintf("操作: replace\n替换了 %d 处匹配\n修改后: %d 行", count, len(lines))

			case "append":
				if params.Content == "" {
					return "", fmt.Errorf("append 操作需要 content 参数")
				}
				// 移除最后一行的空行（如果有）
				if len(lines) > 0 && lines[len(lines)-1] == "" {
					lines = lines[:len(lines)-1]
				}
				lines = append(lines, strings.Split(params.Content, "\n")...)
				result = fmt.Sprintf("操作: append\n追加了 %d 行\n修改后: %d 行", len(strings.Split(params.Content, "\n")), len(lines))

			case "insert":
				if params.Content == "" {
					return "", fmt.Errorf("insert 操作需要 content 参数")
				}
				if params.Line < 1 || params.Line > len(lines)+1 {
					return "", fmt.Errorf("行号超出范围: %d (文件共 %d 行)", params.Line, len(lines))
				}
				insertLines := strings.Split(params.Content, "\n")
				newLines := make([]string, 0, len(lines)+len(insertLines))
				newLines = append(newLines, lines[:params.Line-1]...)
				newLines = append(newLines, insertLines...)
				newLines = append(newLines, lines[params.Line-1:]...)
				lines = newLines
				result = fmt.Sprintf("操作: insert\n在第 %d 行插入了 %d 行\n修改后: %d 行", params.Line, len(insertLines), len(lines))

			case "delete":
				if params.Line < 1 || params.Line > len(lines) {
					return "", fmt.Errorf("行号超出范围: %d (文件共 %d 行)", params.Line, len(lines))
				}
				endLine := params.Line
				if params.EndLine > 0 {
					endLine = params.EndLine
				}
				if endLine < params.Line || endLine > len(lines) {
					return "", fmt.Errorf("结束行号超出范围: %d (文件共 %d 行)", endLine, len(lines))
				}
				deletedCount := endLine - params.Line + 1
				newLines := make([]string, 0, len(lines)-deletedCount)
				newLines = append(newLines, lines[:params.Line-1]...)
				newLines = append(newLines, lines[endLine:]...)
				lines = newLines
				result = fmt.Sprintf("操作: delete\n删除了第 %d-%d 行 (%d 行)\n修改后: %d 行", params.Line, endLine, deletedCount, len(lines))

			default:
				return "", fmt.Errorf("未知操作: %s", params.Action)
			}

			// 写回文件
			if err := os.WriteFile(absPath, []byte(strings.Join(lines, "\n")), 0644); err != nil {
				return "", fmt.Errorf("写回文件失败: %w", err)
			}

			return fmt.Sprintf("文件已编辑: %s\n%s", params.Path, result), nil
		},
	}
}

// newListDirTool 创建列出目录工具
func newListDirTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "list_dir",
				Description: "列出目录内容。显示文件和子目录。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "目录路径（相对于工作目录，默认当前目录）"
						},
						"pattern": {
							"type": "string",
							"description": "文件名匹配模式，如 '*.go'（可选）"
						}
					},
					"required": []
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path    string `json:"path"`
				Pattern string `json:"pattern"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			targetPath := params.Path
			if targetPath == "" {
				targetPath = "."
			}

			absPath, err := validatePath(targetPath)
			if err != nil {
				return "", err
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("目录不存在: %s", targetPath)
				}
				return "", fmt.Errorf("获取目录信息失败: %w", err)
			}

			if !info.IsDir() {
				return "", fmt.Errorf("路径不是目录: %s", targetPath)
			}

			entries, err := os.ReadDir(absPath)
			if err != nil {
				return "", fmt.Errorf("读取目录失败: %w", err)
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("目录内容: %s (%d 项)\n\n", targetPath, len(entries)))

			for _, entry := range entries {
				name := entry.Name()

				if params.Pattern != "" {
					matched, _ := filepath.Match(params.Pattern, name)
					if !matched {
						continue
					}
				}

				if entry.IsDir() {
					result.WriteString(fmt.Sprintf("📁 %s/\n", name))
				} else {
					info, _ := entry.Info()
					result.WriteString(fmt.Sprintf("📄 %s (%s, %s)\n", name, formatSize(info.Size()), info.ModTime().Format("2006-01-02 15:04")))
				}
			}

			return result.String(), nil
		},
	}
}

// newFileInfoTool 创建文件信息工具
func newFileInfoTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "file_info",
				Description: "获取文件或目录的详细信息：大小、权限、修改时间等。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "文件或目录路径（相对于工作目录）"
						}
					},
					"required": ["path"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path string `json:"path"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			absPath, err := validatePath(params.Path)
			if err != nil {
				return "", err
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("文件或目录不存在: %s", params.Path)
				}
				return "", fmt.Errorf("获取文件信息失败: %w", err)
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("文件信息: %s\n", info.Name()))
			result.WriteString(fmt.Sprintf("路径: %s\n", absPath))

			if info.IsDir() {
				result.WriteString("类型: 目录\n")
			} else {
				result.WriteString(fmt.Sprintf("类型: 文件\n大小: %s (%d 字节)\n文本文件: %t\n",
					formatSize(info.Size()), info.Size(), isTextFile(absPath)))
			}

			result.WriteString(fmt.Sprintf("权限: %s (%04o)\n修改时间: %s",
				info.Mode(), info.Mode().Perm(), info.ModTime().Format("2006-01-02 15:04:05")))

			return result.String(), nil
		},
	}
}

// newDeleteFileTool 创建删除文件工具
func newDeleteFileTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "delete_file",
				Description: "删除文件或目录。删除目录需要设置 recursive=true。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"path": {
							"type": "string",
							"description": "文件或目录路径（相对于工作目录）"
						},
						"recursive": {
							"type": "boolean",
							"description": "是否递归删除目录（默认 false）"
						}
					},
					"required": ["path"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Path      string `json:"path"`
				Recursive bool   `json:"recursive"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			absPath, err := validatePath(params.Path)
			if err != nil {
				return "", err
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("文件或目录不存在: %s", params.Path)
				}
				return "", fmt.Errorf("获取文件信息失败: %w", err)
			}

			if info.IsDir() {
				if !params.Recursive {
					entries, err := os.ReadDir(absPath)
					if err != nil {
						return "", fmt.Errorf("读取目录失败: %w", err)
					}
					if len(entries) > 0 {
						return "", fmt.Errorf("目录不为空 (%d 项)，需要设置 recursive=true", len(entries))
					}
				}
				if err := os.RemoveAll(absPath); err != nil {
					return "", fmt.Errorf("删除目录失败: %w", err)
				}
				return fmt.Sprintf("已删除: %s\n类型: 目录", params.Path), nil
			}

			if err := os.Remove(absPath); err != nil {
				return "", fmt.Errorf("删除文件失败: %w", err)
			}
			return fmt.Sprintf("已删除: %s\n类型: 文件", params.Path), nil
		},
	}
}

// newSearchFilesTool 创建按名称搜索文件工具
func newSearchFilesTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "search_files",
				Description: "按文件名模式搜索文件。支持通配符（如 *.go、test_*）。最大返回 100 条结果。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"pattern": {
							"type": "string",
							"description": "文件名匹配模式，支持通配符（如 '*.go'、'test_*'）"
						},
						"path": {
							"type": "string",
							"description": "搜索起始目录（相对于工作目录，默认当前目录）"
						}
					},
					"required": ["pattern"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Pattern string `json:"pattern"`
				Path    string `json:"path"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			targetPath := params.Path
			if targetPath == "" {
				targetPath = "."
			}

			absPath, err := validatePath(targetPath)
			if err != nil {
				return "", err
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("目录不存在: %s", targetPath)
				}
				return "", fmt.Errorf("获取目录信息失败: %w", err)
			}

			if !info.IsDir() {
				return "", fmt.Errorf("路径不是目录: %s", targetPath)
			}

			const maxResults = 100
			var matches []string

			err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
				if err != nil {
					return nil
				}

				if len(matches) >= maxResults {
					return filepath.SkipAll
				}

				if matched, _ := filepath.Match(params.Pattern, d.Name()); matched {
					relPath, _ := filepath.Rel(absPath, path)
					matches = append(matches, relPath)
				}

				return nil
			})
			if err != nil {
				return "", fmt.Errorf("搜索失败: %w", err)
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("搜索结果: '%s' (找到 %d 个文件)\n\n", params.Pattern, len(matches)))

			for _, match := range matches {
				result.WriteString(match + "\n")
			}

			if len(matches) >= maxResults {
				result.WriteString(fmt.Sprintf("\n... 已达到最大结果数 (%d)，可能还有更多匹配", maxResults))
			}

			return result.String(), nil
		},
	}
}

// newSearchContentTool 创建按内容搜索工具
func newSearchContentTool() Tool {
	return Tool{
		Definition: ToolDefinition{
			Type: "function",
			Function: FunctionSchema{
				Name:        "search_content",
				Description: "按文件内容搜索（支持正则表达式）。显示匹配行。最大返回 50 处匹配。",
				Parameters: json.RawMessage(`{
					"type": "object",
					"properties": {
						"query": {
							"type": "string",
							"description": "搜索内容（支持正则表达式）"
						},
						"path": {
							"type": "string",
							"description": "搜索目录（相对于工作目录，默认当前目录）"
						}
					},
					"required": ["query"]
				}`),
			},
		},
		Execute: func(args json.RawMessage) (string, error) {
			var params struct {
				Query string `json:"query"`
				Path  string `json:"path"`
			}
			if err := json.Unmarshal(args, &params); err != nil {
				return "", fmt.Errorf("参数解析失败: %w", err)
			}

			targetPath := params.Path
			if targetPath == "" {
				targetPath = "."
			}

			absPath, err := validatePath(targetPath)
			if err != nil {
				return "", err
			}

			info, err := os.Stat(absPath)
			if err != nil {
				if os.IsNotExist(err) {
					return "", fmt.Errorf("目录不存在: %s", targetPath)
				}
				return "", fmt.Errorf("获取目录信息失败: %w", err)
			}

			if !info.IsDir() {
				return "", fmt.Errorf("路径不是目录: %s", targetPath)
			}

			re, err := regexp.Compile(params.Query)
			if err != nil {
				re = regexp.MustCompile(regexp.QuoteMeta(params.Query))
			}

			const maxMatches = 50
			type match struct {
				File string
				Line int
				Text string
			}
			var matches []match

			err = filepath.WalkDir(absPath, func(path string, d os.DirEntry, err error) error {
				if err != nil || d.IsDir() || !isTextFile(path) || len(matches) >= maxMatches {
					return nil
				}

				content, err := os.ReadFile(path)
				if err != nil {
					return nil
				}

				for i, line := range strings.Split(string(content), "\n") {
					if len(matches) >= maxMatches {
						break
					}
					if re.MatchString(line) {
						relPath, _ := filepath.Rel(absPath, path)
						matches = append(matches, match{File: relPath, Line: i + 1, Text: line})
					}
				}

				return nil
			})
			if err != nil {
				return "", fmt.Errorf("搜索失败: %w", err)
			}

			var result strings.Builder
			result.WriteString(fmt.Sprintf("搜索结果: '%s' (找到 %d 处匹配)\n\n", params.Query, len(matches)))

			for _, m := range matches {
				result.WriteString(fmt.Sprintf("%s:%d\n  %s\n\n", m.File, m.Line, strings.TrimSpace(m.Text)))
			}

			if len(matches) >= maxMatches {
				result.WriteString(fmt.Sprintf("... 已达到最大匹配数 (%d)，可能还有更多匹配", maxMatches))
			}

			return result.String(), nil
		},
	}
}
