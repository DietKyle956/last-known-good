package router

import (
	"context"
	"strings"
)

func DefaultSignals() []string {
	return []string{"debug", "refactor", "complex", "optimize", "architect"}
}

type HeuristicRouter struct {
	MultiFileThreshold int
	ComplexitySignals  []string
	// PostFailureModel controls which model to use after a failure.
	// Empty or "deepseek-v4-pro" routes to Pro with thinking.
	// "deepseek-v4-flash" routes to Flash with thinking enabled.
	PostFailureModel string
}

func (r *HeuristicRouter) Route(_ context.Context, req RouteRequest) RouteDecision {
	if hasComplexitySignal(req.Prompt, r.ComplexitySignals) {
		return RouteDecision{Model: DeepSeekV4Pro, Thinking: true}
	}
	if req.TouchedFiles > r.MultiFileThreshold {
		return RouteDecision{Model: DeepSeekV4Pro, Thinking: true}
	}
	if req.Failed {
		model := r.PostFailureModel
		if model == "" {
			model = DeepSeekV4Pro
		}
		return RouteDecision{Model: model, Thinking: true}
	}
	return RouteDecision{Model: DeepSeekV4Flash, Thinking: false}
}

func hasComplexitySignal(prompt string, signals []string) bool {
	lower := strings.ToLower(prompt)
	for _, s := range signals {
		if strings.Contains(lower, s) {
			return true
		}
	}
	return false
}
