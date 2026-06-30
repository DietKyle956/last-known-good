package router

import "context"

const (
	DeepSeekV4Flash = "deepseek-v4-flash"
	DeepSeekV4Pro   = "deepseek-v4-pro"
)

// RouteRequest contains the context for a routing decision.
type RouteRequest struct {
	TouchedFiles int
	Failed       bool
	Prompt       string
}

// RouteDecision is the result of routing.
type RouteDecision struct {
	Model    string
	Thinking bool
}

// Router decides which model and thinking mode to use for a turn.
type Router interface {
	Route(ctx context.Context, req RouteRequest) RouteDecision
}
