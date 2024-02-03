package external

import (
	"context"
	"testing"
	"time"
)

func TestMockService_Process(t *testing.T) {
	service := MockService{
		limit:     2,
		blockTime: 1 * time.Second,
		rangeTime: 2 * time.Second,
	}

	ctx := context.Background()
	batch := Batch{{Message: "Test1"}, {Message: "Test2"}}

	// Test processing a batch within limit
	err := service.Process(ctx, batch)
	if err != nil {
		t.Errorf("Process() error = %v, wantErr %v", err, false)
	}

	// Test exceeding the limit
	err = service.Process(ctx, batch)
	if err != ErrBlocked {
		t.Errorf("Process() should be blocked but got error = %v", err)
	}

	// Wait for blockTime to pass, with a more generous margin
	time.Sleep(service.rangeTime + 1*time.Second)

	// Test processing after blockTime has passed
	err = service.Process(ctx, batch)
	if err != nil {
		t.Errorf("Process() after block time error = %v, wantErr %v", err, false)
	}

	// Test reset after rangeTime has passed
	time.Sleep(service.rangeTime + 1*time.Second)
	service.requestCount = service.limit // Simulate limit reached before rangeTime reset
	err = service.Process(ctx, Batch{{Message: "Test3"}})
	if err != nil {
		t.Errorf("Process() after rangeTime should reset count and not block, got error = %v", err)
	}
}
