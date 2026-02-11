package usecases

import (
	"context"
	"math/rand/v2"
	"time"

	"github.com/sophialabs/proteusmock/internal/domain/match"
	"github.com/sophialabs/proteusmock/internal/domain/trace"
	"github.com/sophialabs/proteusmock/internal/infrastructure/ports"
	"github.com/sophialabs/proteusmock/internal/infrastructure/services"
)

// HandleRequestResult is the outcome of processing a mock request.
type HandleRequestResult struct {
	Matched     bool
	Response    *match.CompiledResponse
	RateLimited bool
	Pagination  *match.CompiledPagination
	TraceEntry  trace.Entry
}

// HandleRequestUseCase processes incoming mock requests.
type HandleRequestUseCase struct {
	evaluator   *match.Evaluator
	clock       ports.Clock
	rateLimiter ports.RateLimiter
	logger      ports.Logger
	traceBuf    *trace.RingBuffer
}

// NewHandleRequestUseCase creates a new use case.
func NewHandleRequestUseCase(
	evaluator *match.Evaluator,
	clock ports.Clock,
	rateLimiter ports.RateLimiter,
	logger ports.Logger,
	traceBuf *trace.RingBuffer,
) *HandleRequestUseCase {
	return &HandleRequestUseCase{
		evaluator:   evaluator,
		clock:       clock,
		rateLimiter: rateLimiter,
		logger:      logger,
		traceBuf:    traceBuf,
	}
}

// Execute evaluates the request against candidates and returns the result.
func (uc *HandleRequestUseCase) Execute(ctx context.Context, req *match.IncomingRequest, candidates []*match.CompiledScenario) HandleRequestResult {
	evalResult := uc.evaluator.Evaluate(req, candidates)

	entry := trace.Entry{
		Timestamp:  uc.clock.Now(),
		Method:     req.Method,
		Path:       req.Path,
		Candidates: evalResult.Candidates,
	}

	result := HandleRequestResult{
		TraceEntry: entry,
	}

	if evalResult.Matched == nil {
		uc.logger.Debug("no match found", "method", req.Method, "path", req.Path)
		uc.traceBuf.Add(entry)
		return result
	}

	matched := evalResult.Matched
	entry.MatchedID = matched.ID
	result.Matched = true

	// Rate limiting check.
	if matched.Policy != nil && matched.Policy.RateLimit != nil {
		rl := matched.Policy.RateLimit
		key := rl.Key
		if key == "" {
			key = matched.ID
		}
		if !uc.rateLimiter.Allow(ctx, key, rl.Rate, rl.Burst) {
			uc.logger.Debug("rate limited", "scenario", matched.ID, "key", key)
			entry.RateLimited = true
			result.RateLimited = true
			result.TraceEntry = entry
			uc.traceBuf.Add(entry)
			return result
		}
	}

	// Latency simulation (respects context cancellation).
	if matched.Policy != nil && matched.Policy.Latency != nil {
		lat := matched.Policy.Latency
		delay := time.Duration(lat.FixedMs) * time.Millisecond
		if lat.JitterMs > 0 {
			delay += time.Duration(rand.IntN(lat.JitterMs)) * time.Millisecond
		}
		if delay > 0 {
			if err := uc.clock.SleepContext(ctx, delay); err != nil {
				uc.logger.Debug("latency sleep cancelled", "scenario", matched.ID, "error", err)
			}
		}
	}

	resp := matched.Response
	// Infer content type if not explicitly set.
	if resp.ContentType == "" {
		resp.ContentType = services.InferContentType("", "", resp.Body)
	}
	result.Response = &resp

	if matched.Policy != nil && matched.Policy.Pagination != nil {
		result.Pagination = matched.Policy.Pagination
	}

	result.TraceEntry = entry
	uc.traceBuf.Add(entry)

	return result
}
