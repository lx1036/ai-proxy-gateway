package framework

import "github.com/lx1036/gateway/pkg/llmrouter/types"

type SchedulerContext struct {
	Model  string
	Prompt types.ChatMessage
}


