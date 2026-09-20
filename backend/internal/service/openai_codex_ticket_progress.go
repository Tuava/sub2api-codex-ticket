package service

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/google/uuid"
)

const openAICodexTicketProbeProgressTTL = 10 * time.Minute

type OpenAICodexTicketProbeLog struct {
	At       time.Time         `json:"at"`
	Level    string            `json:"level"`
	Event    string            `json:"event"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type OpenAICodexTicketProbeProgress struct {
	OperationID  string                      `json:"operation_id"`
	AccountID    int64                       `json:"account_id"`
	Model        string                      `json:"model"`
	Status       string                      `json:"status"`
	Stage        string                      `json:"stage"`
	Percent      int                         `json:"percent"`
	StartedAt    time.Time                   `json:"started_at"`
	UpdatedAt    time.Time                   `json:"updated_at"`
	CompletedAt  *time.Time                  `json:"completed_at,omitempty"`
	ElapsedMS    int64                       `json:"elapsed_ms"`
	Outcome      string                      `json:"outcome,omitempty"`
	HTTPStatus   int                         `json:"http_status,omitempty"`
	ObservedLen  int                         `json:"observed_length,omitempty"`
	TargetLength int                         `json:"target_length,omitempty"`
	Ready        bool                        `json:"ready"`
	ErrorCode    string                      `json:"error_code,omitempty"`
	ErrorMessage string                      `json:"error_message,omitempty"`
	Logs         []OpenAICodexTicketProbeLog `json:"logs"`
}

type openAICodexTicketProbeProgressState struct {
	mu       sync.RWMutex
	progress OpenAICodexTicketProbeProgress
}

func cloneCodexTicketProbeMetadata(value map[string]string) map[string]string {
	if len(value) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(value))
	for key, item := range value {
		cloned[key] = item
	}
	return cloned
}

func (s *openAICodexTicketProbeProgressState) advance(stage string, percent int, event string, metadata map[string]string) {
	if s == nil {
		return
	}
	now := time.Now()
	if percent < 0 {
		percent = 0
	}
	if percent > 99 {
		percent = 99
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.progress.Status != "running" {
		return
	}
	if percent >= s.progress.Percent {
		s.progress.Percent = percent
	}
	if stage != "" {
		s.progress.Stage = stage
	}
	s.progress.UpdatedAt = now
	if event != "" {
		s.progress.Logs = append(s.progress.Logs, OpenAICodexTicketProbeLog{
			At: now, Level: "info", Event: event, Metadata: cloneCodexTicketProbeMetadata(metadata),
		})
	}
}

func (s *openAICodexTicketProbeProgressState) complete(result *OpenAICodexTicketProbeResult) {
	if s == nil {
		return
	}
	now := time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress.Status = "completed"
	s.progress.Stage = "completed"
	s.progress.Percent = 100
	s.progress.UpdatedAt = now
	s.progress.CompletedAt = &now
	if result != nil {
		s.progress.Outcome = result.Outcome
		s.progress.HTTPStatus = result.HTTPStatus
		s.progress.ObservedLen = result.ObservedLength
		s.progress.TargetLength = result.TargetLength
		s.progress.Ready = result.Ready
	}
	s.progress.Logs = append(s.progress.Logs, OpenAICodexTicketProbeLog{
		At: now, Level: "success", Event: "probe_completed", Metadata: map[string]string{
			"outcome": s.progress.Outcome,
		},
	})
}

func (s *openAICodexTicketProbeProgressState) fail(err error) {
	if s == nil {
		return
	}
	now := time.Now()
	code := infraerrors.Reason(err)
	message := infraerrors.Message(err)
	if message == "" {
		message = "ticket probe failed"
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.progress.Status = "failed"
	s.progress.Stage = "failed"
	s.progress.Percent = 100
	s.progress.UpdatedAt = now
	s.progress.CompletedAt = &now
	s.progress.ErrorCode = code
	s.progress.ErrorMessage = message
	s.progress.Logs = append(s.progress.Logs, OpenAICodexTicketProbeLog{
		At: now, Level: "error", Event: "probe_failed", Metadata: map[string]string{
			"code": code,
		},
	})
}

func (s *openAICodexTicketProbeProgressState) snapshot() *OpenAICodexTicketProbeProgress {
	if s == nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	value := s.progress
	value.Logs = append([]OpenAICodexTicketProbeLog(nil), s.progress.Logs...)
	for i := range value.Logs {
		value.Logs[i].Metadata = cloneCodexTicketProbeMetadata(value.Logs[i].Metadata)
	}
	end := time.Now()
	if value.CompletedAt != nil {
		end = *value.CompletedAt
	}
	value.ElapsedMS = end.Sub(value.StartedAt).Milliseconds()
	return &value
}

func (s *OpenAIGatewayService) beginOpenAICodexTicketProbeProgress(operationID string, accountID int64, model string) (*openAICodexTicketProbeProgressState, error) {
	if s == nil {
		return nil, infraerrors.ServiceUnavailable("CODEX_TICKET_PROBER_UNAVAILABLE", "codex ticket service unavailable")
	}
	if _, err := uuid.Parse(operationID); err != nil {
		return nil, infraerrors.BadRequest("CODEX_TICKET_OPERATION_INVALID", "operation_id must be a UUID")
	}
	now := time.Now()
	state := &openAICodexTicketProbeProgressState{progress: OpenAICodexTicketProbeProgress{
		OperationID: operationID,
		AccountID:   accountID,
		Model:       model,
		Status:      "running",
		Stage:       "queued",
		Percent:     1,
		StartedAt:   now,
		UpdatedAt:   now,
		Logs: []OpenAICodexTicketProbeLog{{
			At: now, Level: "info", Event: "probe_queued",
		}},
	}}
	if _, loaded := s.openaiCodexTicketProbeRuns.LoadOrStore(operationID, state); loaded {
		return nil, infraerrors.Conflict("CODEX_TICKET_OPERATION_EXISTS", "operation_id is already in use")
	}
	time.AfterFunc(openAICodexTicketProbeProgressTTL, func() {
		s.openaiCodexTicketProbeRuns.CompareAndDelete(operationID, state)
	})
	return state, nil
}

func (s *OpenAIGatewayService) GetOpenAICodexTicketProbeProgress(accountID int64, operationID string) (*OpenAICodexTicketProbeProgress, error) {
	if s == nil {
		return nil, infraerrors.ServiceUnavailable("CODEX_TICKET_PROBER_UNAVAILABLE", "codex ticket service unavailable")
	}
	raw, ok := s.openaiCodexTicketProbeRuns.Load(operationID)
	if !ok {
		return nil, infraerrors.NotFound("CODEX_TICKET_OPERATION_NOT_FOUND", "ticket probe operation was not found")
	}
	state, ok := raw.(*openAICodexTicketProbeProgressState)
	if !ok || state == nil {
		return nil, infraerrors.InternalServer("CODEX_TICKET_OPERATION_INVALID_STATE", "ticket probe progress is unavailable")
	}
	progress := state.snapshot()
	if progress == nil || progress.AccountID != accountID {
		return nil, infraerrors.NotFound("CODEX_TICKET_OPERATION_NOT_FOUND", "ticket probe operation was not found")
	}
	return progress, nil
}

func codexTicketProbeResponseMetadata(status, length, target int) map[string]string {
	return map[string]string{
		"http_status":     strconv.Itoa(status),
		"observed_length": strconv.Itoa(length),
		"target_length":   strconv.Itoa(target),
	}
}

func codexTicketProbePolicyMetadata(model string, target int) map[string]string {
	return map[string]string{"model": model, "target_length": fmt.Sprintf("%d", target)}
}
