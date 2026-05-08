package pd

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io"
	"k8s.io/klog/v2"
	"maps"
	"net/http"
)

// KVTransferParams holds parameters for Key-Value cache transfer between generation steps.
type KVTransferParams struct {
	DoRemoteDecode  bool     `json:"do_remote_decode"`
	DoRemotePrefill bool     `json:"do_remote_prefill"`
	RemoteEngineID  *string  `json:"remote_engine_id,omitempty"`
	RemoteBlockIDs  []string `json:"remote_block_ids,omitempty"`
	RemoteHost      *string  `json:"remote_host,omitempty"`
	RemotePort      *int     `json:"remote_port,omitempty"`
}

type NIXLConnector struct {
	name string
}

func NewNixlConnector() KVConnector {
	return &NIXLConnector{
		name: "nixl",
	}
}

func (connector *NIXLConnector) Proxy(c *gin.Context, modelRequest map[string]interface{}, prefillEndpoint, decodeEndpoint string) {

	prefillRequest := connector.buildPrefillRequest(c.Request, modelRequest)

	// INFO: 1. prefill with kv_transfer_params
	kvTransferParams, err := connector.prefill(prefillRequest, prefillEndpoint)
	if err != nil {

	}

	// INFO: 2. decode
	decodeRequest := connector.buildDecodeRequest(c.Request, modelRequest, kvTransferParams)

	err = connector.decode(c, decodeRequest, decodeEndpoint)
	if err != nil {

	}
}


func (connector *NIXLConnector) buildPrefillRequest(req *http.Request, modelRequest map[string]interface{}) *http.Request {
	var reqBody map[string]interface{}
	maps.Copy(reqBody, modelRequest)
	delete(reqBody, "stream")
	delete(reqBody, "stream_options")
	reqBody["max_tokens"] = 1 // Prefill Request max_tokens=1
	if _, ok := reqBody["max_completion_tokens"]; ok {
		reqBody["max_completion_tokens"] = 1
	}

	// Add NIXL-specific parameters for KV cache transfer.
	reqBody["kv_transfer_params"] = &KVTransferParams{
		DoRemoteDecode:  true,
		DoRemotePrefill: false,
	}
	bodyBytes, _ := json.Marshal(reqBody)

	reqCopy := req.Clone(req.Context())
	reqCopy.URL.Scheme = "http"
	reqCopy.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	reqCopy.ContentLength = int64(len(bodyBytes))
	return reqCopy
}

func (connector *NIXLConnector) buildDecodeRequest(req *http.Request, modelRequest map[string]interface{}, kvTransferParams interface{}) *http.Request {
	var reqBody map[string]interface{}
	maps.Copy(reqBody, modelRequest)

	// INFO: decode request 需要加上 "kv_transfer_params"
	reqBody["kv_transfer_params"] = kvTransferParams
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		return nil
	}

	reqCopy := req.Clone(req.Context())
	reqCopy.URL.Scheme = "http"
	reqCopy.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
	reqCopy.ContentLength = int64(len(bodyBytes))
	return reqCopy
}



func (connector *NIXLConnector) prefill(prefillReq *http.Request, prefillEndpoint string) (interface{}, error)  {
	prefillReq.URL.Host = prefillEndpoint
	prefillReq.URL.Scheme = "http"
	resp, err := http.DefaultTransport.RoundTrip(prefillReq)
	if err != nil {
		return nil, fmt.Errorf("do request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("http response error, unexpected status code: %d", resp.StatusCode)
	}

	body ,err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body error: %w", err)
	}

	var prefillResponse map[string]interface{}
	err = json.Unmarshal(body, &prefillResponse)
	if err != nil {
		return nil, fmt.Errorf("json unmarshal error: %w", err)
	}


	kvTransferParams, ok := prefillResponse["kv_transfer_params"]
	if !ok {
		klog.Warning("NIXL: missing 'kv_transfer_params' in prefill response")
	}
	klog.Infof("NIXL: kv_transfer_params: %v", kvTransferParams)

	return kvTransferParams, nil
}

func (connector *NIXLConnector) decode(c *gin.Context, decodeReq *http.Request, decodeEndpoint string) error {
	decodeReq.URL.Host = decodeEndpoint
	decodeReq.URL.Scheme = "http"
	resp, err := http.DefaultTransport.RoundTrip(decodeReq)
	if err != nil {
		return fmt.Errorf("do request error: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("http response error, unexpected status code: %d", resp.StatusCode)
	}

	for key, value := range resp.Header {
		for _, v := range value {
			c.Header(key, v)
		}
	}

	c.Status(resp.StatusCode)


	// Determine if this is a streaming response
	stream := isStreamingResponse(resp)
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


