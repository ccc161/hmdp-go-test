package utils

import (
	"github.com/go-resty/resty/v2"
	"github.com/spf13/viper"
	"io"
	"net"
	"net/http"
	"syscall"
	"time"
)

func InitHttpClient() *resty.Client {
	transport := &http.Transport{
		MaxIdleConns:          800,              // 提高全局空闲连接数
		MaxIdleConnsPerHost:   400,              // 关键参数：每个Host的空闲连接
		MaxConnsPerHost:       1000,             // 限制对同一Host的总连接数
		IdleConnTimeout:       60 * time.Second, // 延长空闲连接保留时间
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,   // 连接建立超时
			KeepAlive: 60 * time.Second,   // 保持心跳间隔
			Control:   reusePortControl(), // 启用端口复用
		}).DialContext,
	}

	client := resty.NewWithClient(&http.Client{
		Transport: transport,
		Timeout:   30 * time.Second, // 全局超时覆盖（包含连接+响应）
	}).
		SetRetryCount(2). // 简单重试机制
		SetRetryWaitTime(1 * time.Second).
		SetBaseURL(viper.GetString("api.base_url")).
		SetHeaders(map[string]string{
			"User-Agent":   "Apifox/1.0.0 (https://apifox.com)",
			"Accept":       "*/*",
			"Connection":   "keep-alive",
			"Content-Type": "application/json",
		}).
		OnAfterResponse(func(c *resty.Client, resp *resty.Response) error {
			_, _ = io.Copy(io.Discard, resp.RawBody()) // 确保响应体被读取
			return nil
		})

	return client
}

func reusePortControl() func(network, addr string, c syscall.RawConn) error {
	return func(network, addr string, c syscall.RawConn) error {
		return c.Control(func(fd uintptr) {
			syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
		})
	}
}
