package utils

import (
	"github.com/gomarkdown/markdown"
	"github.com/gomarkdown/markdown/html"
	"github.com/gomarkdown/markdown/parser"
)

// MarkdownToHTML 将Markdown转换为HTML
func MarkdownToHTML(md string) string {
	// 创建解析器
	extensions := parser.CommonExtensions
	parser := parser.NewWithExtensions(extensions)

	// 解析Markdown
	doc := parser.Parse([]byte(md))

	// 创建HTML渲染器
	htmlFlags := html.CommonFlags | html.HrefTargetBlank
	opts := html.RendererOptions{Flags: htmlFlags}
	renderer := html.NewRenderer(opts)

	// 渲染为HTML
	html := markdown.Render(doc, renderer)

	return string(html)
}

// SanitizeMarkdown 清理Markdown内容
func SanitizeMarkdown(md string) string {
	// 这里可以添加Markdown内容清理逻辑
	// 例如移除危险标签、限制长度等
	return md
}
