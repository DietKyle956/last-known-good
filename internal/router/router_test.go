package router

import (
	"context"
	"testing"
)

func TestSingleFileRoutesToFlash(t *testing.T) {
	r := &HeuristicRouter{MultiFileThreshold: 2, ComplexitySignals: DefaultSignals()}
	req := RouteRequest{TouchedFiles: 1}
	dec := r.Route(context.Background(), req)
	if dec.Model != DeepSeekV4Flash {
		t.Errorf("expected model %q, got %q", DeepSeekV4Flash, dec.Model)
	}
	if dec.Thinking {
		t.Error("expected thinking disabled for single-file Flash turn")
	}
}

func TestMultiFileRoutesToPro(t *testing.T) {
	r := &HeuristicRouter{MultiFileThreshold: 2, ComplexitySignals: DefaultSignals()}
	req := RouteRequest{TouchedFiles: 3}
	dec := r.Route(context.Background(), req)
	if dec.Model != DeepSeekV4Pro {
		t.Errorf("expected model %q, got %q", DeepSeekV4Pro, dec.Model)
	}
	if !dec.Thinking {
		t.Error("expected thinking enabled for Pro turn")
	}
}

func TestPostFailureRoutesToPro(t *testing.T) {
	r := &HeuristicRouter{MultiFileThreshold: 2, ComplexitySignals: DefaultSignals()}
	req := RouteRequest{Failed: true, TouchedFiles: 1}
	dec := r.Route(context.Background(), req)
	if dec.Model != DeepSeekV4Pro {
		t.Errorf("expected model %q after failure, got %q", DeepSeekV4Pro, dec.Model)
	}
	if !dec.Thinking {
		t.Error("expected thinking enabled for post-failure Pro turn")
	}
}

func TestComplexitySignalRoutesToPro(t *testing.T) {
	r := &HeuristicRouter{MultiFileThreshold: 2, ComplexitySignals: DefaultSignals()}
	req := RouteRequest{Prompt: "debug this issue", TouchedFiles: 1}
	dec := r.Route(context.Background(), req)
	if dec.Model != DeepSeekV4Pro {
		t.Errorf("expected model %q for complexity signal, got %q", DeepSeekV4Pro, dec.Model)
	}
	if !dec.Thinking {
		t.Error("expected thinking enabled for complexity signal Pro turn")
	}
}

func TestFlashRetryHasThinkingEnabled(t *testing.T) {
	r := &HeuristicRouter{
		MultiFileThreshold: 2,
		ComplexitySignals:  DefaultSignals(),
		PostFailureModel:   DeepSeekV4Flash,
	}
	req := RouteRequest{Failed: true, TouchedFiles: 1}
	dec := r.Route(context.Background(), req)
	if dec.Model != DeepSeekV4Flash {
		t.Errorf("expected model %q for Flash retry, got %q", DeepSeekV4Flash, dec.Model)
	}
	if !dec.Thinking {
		t.Error("expected thinking enabled for Flash retry turn")
	}
}

func TestRouterIsPluggable(t *testing.T) {
	var r Router = &stubRouter{decision: RouteDecision{Model: DeepSeekV4Pro, Thinking: true}}
	dec := r.Route(context.Background(), RouteRequest{TouchedFiles: 1})
	if dec.Model != DeepSeekV4Pro {
		t.Errorf("expected stub to return Pro, got %q", dec.Model)
	}
	if !dec.Thinking {
		t.Error("expected stub to return thinking enabled")
	}
}

type stubRouter struct {
	decision RouteDecision
}

func (s *stubRouter) Route(_ context.Context, _ RouteRequest) RouteDecision {
	return s.decision
}
