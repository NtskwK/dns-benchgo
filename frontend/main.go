package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"golang.org/x/net/html"
)

const distRoot = "./web/dist"

func getProcessedHTML() (string, error) {
	// 读取index.html文件
	htmlPath := filepath.Join(distRoot, "index.html")
	content, err := ioutil.ReadFile(htmlPath)
	if err != nil {
		return "", fmt.Errorf("无法读取HTML文件: %v", err)
	}

	// 解析HTML
	doc, err := html.Parse(bytes.NewReader(content))
	if err != nil {
		return "", fmt.Errorf("解析HTML失败: %v", err)
	}

	// 查找head和body元素
	var head, body *html.Node
	var findNodes func(*html.Node)
	findNodes = func(n *html.Node) {
		if n.Type == html.ElementNode {
			if n.Data == "head" {
				head = n
			} else if n.Data == "body" {
				body = n
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			findNodes(c)
		}
	}
	findNodes(doc)

	if head == nil {
		fmt.Println("未找到head元素")
		return string(content), nil
	}

	if body == nil {
		fmt.Println("未找到body元素")
		return string(content), nil
	}

	// 收集并处理script标签
	scripts := collectScripts(head)
	fmt.Println("head中的脚本:", scripts)

	// 读取脚本内容并添加到body
	for _, src := range scripts {
		err := embedScript(body, src)
		if err != nil {
			fmt.Printf("处理脚本 %s 时出错: %v\n", src, err)
		}
	}

	// 收集并处理link标签中的CSS
	stylesheets := collectStylesheets(head)
	fmt.Println("head中的CSS文件:", stylesheets)

	// 读取CSS内容并添加到head
	for _, href := range stylesheets {
		err := embedStylesheet(head, href)
		if err != nil {
			fmt.Printf("处理CSS %s 时出错: %v\n", href, err)
		}
	}

	// 将处理后的HTML转换为字符串
	var buf bytes.Buffer
	if err := html.Render(&buf, doc); err != nil {
		return "", fmt.Errorf("渲染HTML失败: %v", err)
	}

	return buf.String(), nil
}

// 收集head中的所有script标签的src属性
func collectScripts(head *html.Node) []string {
	var scripts []string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" {
			for _, attr := range n.Attr {
				if attr.Key == "src" && attr.Val != "" {
					scripts = append(scripts, attr.Val)
					break
				}
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(head)
	return scripts
}

// 收集head中的所有stylesheet链接
func collectStylesheets(head *html.Node) []string {
	var stylesheets []string
	var traverse func(*html.Node)
	traverse = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "link" {
			var isStylesheet bool
			var href string
			for _, attr := range n.Attr {
				if attr.Key == "rel" && attr.Val == "stylesheet" {
					isStylesheet = true
				}
				if attr.Key == "href" && attr.Val != "" {
					href = attr.Val
				}
			}
			if isStylesheet && href != "" {
				stylesheets = append(stylesheets, href)
			}
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			traverse(c)
		}
	}
	traverse(head)
	return stylesheets
}

// 读取脚本文件并嵌入到body中
func embedScript(body *html.Node, src string) error {
	// 处理路径
	scriptPath := filepath.Join(distRoot, filepath.Clean(src))
	
	// 读取脚本内容
	content, err := ioutil.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("无法读取脚本文件: %v", err)
	}

	// 创建新的script标签
	scriptNode := &html.Node{
		Type: html.ElementNode,
		Data: "script",
	}
	
	// 添加脚本内容
	textNode := &html.Node{
		Type: html.TextNode,
		Data: string(content),
	}
	scriptNode.AppendChild(textNode)
	
	// 添加到body
	body.AppendChild(scriptNode)
	
	fmt.Printf("成功加载脚本: %s\n", scriptPath)
	return nil
}

// 读取CSS文件并嵌入到head中
func embedStylesheet(head *html.Node, href string) error {
	// 处理路径
	cssPath := filepath.Join(distRoot, filepath.Clean(href))
	
	// 读取CSS内容
	content, err := ioutil.ReadFile(cssPath)
	if err != nil {
		return fmt.Errorf("无法读取CSS文件: %v", err)
	}

	// 创建新的style标签
	styleNode := &html.Node{
		Type: html.ElementNode,
		Data: "style",
	}
	
	// 添加CSS内容
	textNode := &html.Node{
		Type: html.TextNode,
		Data: string(content),
	}
	styleNode.AppendChild(textNode)
	
	// 添加到head
	head.AppendChild(styleNode)
	
	fmt.Printf("成功加载CSS: %s\n", cssPath)
	return nil
}

// 处理根路径请求
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	htmlContent, err := getProcessedHTML()
	if err != nil {
		http.Error(w, fmt.Sprintf("处理HTML时出错: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, htmlContent)
}

func main() {
	http.HandleFunc("/", homeHandler)
	
	fmt.Println("服务器启动在: http://localhost:8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Printf("服务器启动失败: %v\n", err)
		os.Exit(1)
	}
}
