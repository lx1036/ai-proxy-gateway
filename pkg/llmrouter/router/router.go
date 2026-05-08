package router

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lx1036/gateway/pkg/llmrouter/router/pd"
	"io"
	"k8s.io/klog/v2"
	"net/http"
	"os"
	"strconv"
)

type ModelRequest map[string]interface{}

type Router struct {
}

func NewRouter() *Router {
	return &Router{}
}

func (r *Router) HandlerFunc() gin.HandlerFunc {
	// TODO: access log middleware
	return func(c *gin.Context) {

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}

		var modelRequest ModelRequest
		err = json.Unmarshal(bodyBytes, &modelRequest)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, err)
			return
		}

		modelName, ok := modelRequest["model"]
		if !ok {
			c.AbortWithStatusJSON(http.StatusBadRequest, "model name is required")
			return
		}

		klog.Infof("model name: %s", modelName)

		/*
			// INFO: 读取了一次后，c.Request.Body 就为空了
			bodyBytes2, err := io.ReadAll(c.Request.Body)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, err)
				return
			}
			klog.Infof("bodyBytes2: %s", string(bodyBytes2))*/

		// Store model name in context for metrics middleware
		c.Set("model", modelName)

		// TODO: rate limit for modelName

		requestID := uuid.New().String()
		if c.Request.Header.Get("x-request-id") == "" {
			c.Request.Header.Set("x-request-id", requestID)
		}

		if !EnableFairnessScheduling {
			r.scheduling(c, modelRequest)
			return
		}

		// load balance for fairness scheduling
		if err = r.fairnessScheduling(); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, err)
			return
		}
	}
}

func (r *Router) scheduling(c *gin.Context, modelRequest ModelRequest) {
	//err = r.scheduler.Schedule(ctx, pods)
	// TODO: schedule pods -> llm plugins -> target pod
	podIP := "localhost"
	port := 18080

	endpoint := fmt.Sprintf("%s:%d", podIP, port)
	klog.Infof("proxy model endpoint: %s", endpoint)

	stream := false
	if v, ok := modelRequest["stream"]; ok {
		stream = v.(bool)
	}

	bodyBytes, _ := json.Marshal(modelRequest)
	c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	c.Request.ContentLength = int64(len(bodyBytes))
	// INFO: 1. 非 PD 分离场景
	err := r.proxyModelEndpoint(c, endpoint, stream)
	if err != nil {
		klog.Errorf("proxy model endpoint error: %v", err)
		c.AbortWithStatusJSON(http.StatusInternalServerError, err)
		return
	}

	// INFO: 2. PD 分离场景
	prefillEndpoint := "localhost:18080"
	decodeEndpoint := "localhost:18081"

	r.proxyPrefillDecode(c, modelRequest, prefillEndpoint, decodeEndpoint)
}

func (r *Router) proxyPrefillDecode(c *gin.Context, modelRequest ModelRequest, prefillEndpoint, decodeEndpoint string) {

	connectorName := "lmcache"
	connector := pd.GetConnector(connectorName)
	connector().Proxy(c, modelRequest, prefillEndpoint, decodeEndpoint)

}

func (r *Router) fairnessScheduling() error {
	return nil
}

func (r *Router) proxyModelEndpoint(c *gin.Context, endpoint string, stream bool) error {

	resp, err := doRequest(c.Request, endpoint)
	if err != nil {
		return fmt.Errorf("decode request error: %w", err)
	}

	// map[Content-Length:[2857] Content-Type:[application/json] Date:[Tue, 21 Apr 2026 03:14:37 GMT] Server:[uvicorn]]
	klog.Infof("response headers: %+v", resp.Header)

	for key, value := range resp.Header {
		for _, v := range value {
			c.Header(key, v)
		}
	}
	defer resp.Body.Close()

	c.Status(resp.StatusCode)

	if stream {
		reader := bufio.NewReader(resp.Body)
		c.Stream(func(w io.Writer) bool {
			// 协议格式识别：\n 表示按行读取流式响应，这是典型的 Server-Sent Events (SSE) 或类似流式传输协议格式
			// 数据分块处理：在 LLM/AI 代理网关场景中，模型响应通常以换行符分隔的 JSON 片段形式返回
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					klog.Errorf("read response stream body error: %v", err)
				}

				return false
			}

			// data: {"id":"chatcmpl-c52062b1-c8bd-4cb6-8a0b-ed7e058dc29e","object":"chat.completion.chunk","created":1776742036,"model":"qwen","choices":[{"index":0,"delta":{"content":"告诉我"},"logprobs":null,"finish_reason":null,"token_ids":null}]}
			_, err = w.Write(line)
			if err != nil {
				klog.Errorf("write response body error: %v", err)
				return false
			}

			return true
		})
	} else {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return fmt.Errorf("read response body error: %w", err)
		}

		_, err = c.Writer.Write(bodyBytes)
		if err != nil {
			return fmt.Errorf("write response body error: %w", err)
		}
	}

	return nil
}

func doRequest(req *http.Request, endpoint string) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = endpoint
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, fmt.Errorf("do request error: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http response error, unexpected status code: %d", resp.StatusCode)
	}

	return resp, nil
}

var EnableFairnessScheduling = getEnvBool("ENABLE_FAIRNESS_SCHEDULING", false)

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}
