package usage

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/uniedit/server/internal/module/billing"
	"github.com/uniedit/server/internal/module/billing/domain"
	"go.uber.org/zap"
)

// Record represents a usage record to be persisted.
type Record struct {
	UserID       uuid.UUID
	RequestID    string
	TaskType     string
	ProviderID   uuid.UUID
	ModelID      string
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	CostUSD      float64
	LatencyMs    int
	Success      bool
	Timestamp    time.Time
}

// Recorder records usage asynchronously.
type Recorder struct {
	repo    billing.Repository
	logger  *zap.Logger
	buffer  chan *Record
	wg      sync.WaitGroup
	done    chan struct{}
}

// NewRecorder creates a new usage recorder.
func NewRecorder(repo billing.Repository, logger *zap.Logger, bufferSize int) *Recorder {
	if bufferSize <= 0 {
		bufferSize = 1000
	}
	r := &Recorder{
		repo:   repo,
		logger: logger,
		buffer: make(chan *Record, bufferSize),
		done:   make(chan struct{}),
	}
	r.start()
	return r
}

// Record queues a usage record for persistence.
func (r *Recorder) Record(record *Record) {
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	select {
	case r.buffer <- record:
		// Successfully queued
	default:
		// Buffer full, log and drop
		r.logger.Warn("usage record buffer full, dropping record",
			zap.String("request_id", record.RequestID),
		)
	}
}

// Close stops the recorder and flushes remaining records.
func (r *Recorder) Close() {
	close(r.done)
	r.wg.Wait()
}

func (r *Recorder) start() {
	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		for {
			select {
			case record := <-r.buffer:
				r.persist(record)
			case <-r.done:
				// Flush remaining records
				for {
					select {
					case record := <-r.buffer:
						r.persist(record)
					default:
						return
					}
				}
			}
		}
	}()
}

func (r *Recorder) persist(record *Record) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	usageRecord := domain.NewUsageRecord(
		record.UserID,
		record.RequestID,
		record.TaskType,
		record.ProviderID,
		record.ModelID,
		record.InputTokens,
		record.OutputTokens,
		record.CostUSD,
		record.LatencyMs,
		record.Success,
	)

	if err := r.repo.CreateUsageRecord(ctx, usageRecord); err != nil {
		r.logger.Error("failed to persist usage record",
			zap.Error(err),
			zap.String("request_id", record.RequestID),
		)
	}
}
