package client

import (
	service "awesomeProject/external"
	"context"
	"testing"
	"time"
)

type MockService struct {
	ProcessFunc   func(ctx context.Context, batch service.Batch) error
	GetLimitsFunc func() (uint64, time.Duration)
}

func (m *MockService) Process(ctx context.Context, batch service.Batch) error {
	return m.ProcessFunc(ctx, batch)
}

func (m *MockService) GetLimits() (uint64, time.Duration) {
	return m.GetLimitsFunc()
}

func TestProcessBatch(t *testing.T) {
	// Mock service setup
	mockService := &MockService{
		ProcessFunc: func(ctx context.Context, batch service.Batch) error {
			// Simulate successful processing
			return nil
		},
		GetLimitsFunc: func() (uint64, time.Duration) {
			return 2, 1 * time.Second // Example: limit of 2 items per 1 second
		},
	}

	// Client setup with the mock service
	c := NewClient(mockService)

	// Enqueue more items than the limit to test batching
	c.EnqueueItems([]service.Item{{}, {}, {}, {}}) // Enqueue 4 items

	// Allow some time for processing
	time.Sleep(2 * time.Second) // Adjust based on your service's timing
}

func TestService_ProcessBlocked(t *testing.T) {
	// Creating a service instance that is initially blocked
	testService := service.NewTestService()

	ctx := context.Background()
	batch := service.Batch{{Message: "Test1"}}

	// Attempt to process a batch while the service is initially blocked
	err := testService.Process(ctx, batch)
	if err != service.ErrBlocked {
		t.Errorf("Expected ErrBlocked, got %v", err)
	}
}
