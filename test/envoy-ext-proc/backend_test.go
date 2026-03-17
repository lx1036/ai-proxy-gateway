package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"testing"
	"time"
)

// 处理根路径 / 的请求
func rootHandler(w http.ResponseWriter, r *http.Request) {
	// 记录请求信息（方法、路径、客户端IP）
	log.Printf("收到请求: %s %s, 客户端IP: %s", r.Method, r.URL.Path, r.RemoteAddr)
	// 设置响应头（指定UTF-8编码，避免中文乱码）
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	// 响应内容
	_, err := w.Write([]byte("欢迎访问根路径 /\n这是未受保护的公开接口"))
	if err != nil {
		log.Printf("响应写入失败: %v", err)
	}
}

// 处理 /unprotected 路径的请求
func unprotectedHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("收到请求: %s %s, 客户端IP: %s", r.Method, r.URL.Path, r.RemoteAddr)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, err := w.Write([]byte("欢迎访问 /unprotected 路径\n这是未受保护的公开接口"))
	if err != nil {
		log.Printf("响应写入失败: %v", err)
	}
}

func TestBackend(test *testing.T) {
	// 注册路由：将路径映射到对应的处理函数
	http.HandleFunc("/", rootHandler)                   // 根路径
	http.HandleFunc("/unprotected", unprotectedHandler) // /unprotected 路径

	// 配置HTTP服务器
	server := &http.Server{
		Addr:         ":9099",          // 监听9099端口（所有网卡）
		ReadTimeout:  10 * time.Second, // 读取超时
		WriteTimeout: 10 * time.Second, // 写入超时
		IdleTimeout:  15 * time.Second, // 空闲连接超时
	}

	// 启动HTTP服务器（异步启动，避免阻塞后续的优雅退出逻辑）
	go func() {
		log.Printf("HTTP服务器已启动，监听端口: 9099")
		log.Printf("可访问: http://localhost:9099/")
		log.Printf("可访问: http://localhost:9099/unprotected")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 优雅退出：监听系统信号（Ctrl+C、kill 命令）
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT（中断，Ctrl+C）和 SIGTERM（终止，kill）信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit // 阻塞，直到收到退出信号

	log.Println("开始关闭HTTP服务器...")
	// 关闭服务器（5秒超时）
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("服务器强制关闭: %v", err)
	}
	log.Println("服务器已优雅关闭")
}
