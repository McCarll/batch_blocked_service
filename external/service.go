package external

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

// ErrBlocked using for providing exception if service is blocked
var ErrBlocked = errors.New("blocked")

type Service interface {
	GetLimits() (n uint64, p time.Duration)
	Process(ctx context.Context, batch Batch) error
}

type Batch []Item
type Item struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

type MockService struct {
	limit        uint64        // maximum number of items that can be processed
	rangeTime    time.Duration // time window for limit enforcement
	blockTime    time.Duration // duration to block service if limit is exceeded
	lastReset    time.Time     // last time the request count was reset
	requestCount uint64        // count of processed items in the current time window
	mu           sync.Mutex    // mutex for concurrent access to service state
	blockUntil   time.Time     // time then service will be available
}

func NewMockService(limit uint64, rangeTime, blockTime time.Duration) *MockService {
	return &MockService{
		limit:      limit,
		rangeTime:  rangeTime,
		blockTime:  blockTime,
		lastReset:  time.Now(),
		blockUntil: time.Now().Add(-10 * time.Second),
	}
}
func NewTestService() *MockService {
	return &MockService{
		limit:      0,
		rangeTime:  0,
		blockTime:  0,
		lastReset:  time.Now(),
		blockUntil: time.Now().Add(10 * time.Minute),
	}
}

func (m *MockService) GetLimits() (n uint64, p time.Duration) {
	return m.limit, m.rangeTime
}

func (m *MockService) Process(ctx context.Context, batch Batch) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	if now.Before(m.blockUntil) {
		return ErrBlocked
	}

	if now.Sub(m.lastReset) > m.rangeTime {
		m.requestCount = 0
		m.lastReset = now
	}

	if m.requestCount+uint64(len(batch)) > m.limit {
		m.blockUntil = now.Add(m.blockTime)
		return ErrBlocked
	}

	m.requestCount += uint64(len(batch))
	time.Sleep(100 * time.Millisecond)
	for _, item := range batch {
		fmt.Printf("Processing Item: %s\n", item.Message)
	}

	return nil
}
