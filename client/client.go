package client

import (
	service "awesomeProject/external"
	"context"
	"log"
	"sync"
	"time"
)

// Client manages batches of service items, ensuring they are processed
// according to service limits and handling retries if the service is blocked.
type Client struct {
	service       service.Service
	queue         []service.Item
	queueLimit    uint64
	processTimer  *time.Timer
	mu            sync.Mutex
	lastBatchTime time.Time
}

func NewClient(srv service.Service) *Client {
	return &Client{
		service: srv,
	}
}

func (c *Client) EnqueueItems(items []service.Item) {
	c.mu.Lock()
	queueSize := len(c.queue)
	c.queue = append(c.queue, items...)
	currentQueueSize := len(c.queue)
	c.mu.Unlock()

	if queueSize > 0 {
		return
	}
	// If the queue reaches the batch limit, process it immediately.
	// Otherwise, if the processTimer is nil (no timer is set), start the timer.
	if uint64(currentQueueSize) >= c.queueLimit {
		go c.processBatch()
	} else if c.processTimer == nil {
		go c.processQueue()
	}
}

func (c *Client) processQueue() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// If the queue size is smaller than the batch limit and the timer is not set, start it.
	if uint64(len(c.queue)) < c.queueLimit && c.processTimer == nil {
		_, p := c.service.GetLimits()
		c.processTimer = time.AfterFunc(p, func() {
			// Ensure the timer is cleared before calling processBatch
			c.mu.Lock()
			c.processTimer = nil
			c.mu.Unlock()
			c.processBatch()
		})
	}
}

func (c *Client) startProcessingTimer() {
	_, p := c.service.GetLimits()
	// Timer is already stopped or nil, safe to replace
	c.processTimer = time.AfterFunc(p, func() {
		// This function will be called in its own goroutine when the timer fires
		c.processQueue()
	})
}

func (c *Client) processBatch() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Get current time to calculate if the SERVICE_PERIOD has elapsed since the last batch
	now := time.Now()

	limit, period := c.service.GetLimits()

	// Calculate the time since the last batch was sent
	timeSinceLastBatch := now.Sub(c.lastBatchTime)

	// Calculate the correct batch size respecting the limit
	batchSize := int(limit)
	if len(c.queue) < batchSize {
		batchSize = len(c.queue)
	}

	// If there's nothing to process, return early
	if batchSize == 0 {
		return
	}

	// Check if the SERVICE_PERIOD has elapsed since the last batch, or if the queue size is less than the limit
	// which allows processing another batch immediately
	if timeSinceLastBatch >= period || len(c.queue) < int(limit) {
		// Prepare the batch respecting the limit
		batch := make(service.Batch, 0, batchSize)
		batch, c.queue = c.queue[:batchSize], c.queue[batchSize:]

		// Process the batch outside of the lock to allow other operations to proceed
		c.mu.Unlock()
		err := c.service.Process(context.Background(), batch)
		c.mu.Lock()

		if err != nil {
			log.Printf("Error processing batch: %v", err)
			if err == service.ErrBlocked {
				// If the service is blocked, requeue the batch and don't update the lastBatchTime
				c.queue = append(batch, c.queue...)
			}
			// Handle other types of errors as needed
		} else {
			log.Printf("Processed a batch of %d items successfully.", batchSize)
			// Update the last processed time only if the batch was successfully sent
			c.lastBatchTime = now

			// If the batch was smaller than the limit, process the next batch immediately without updating the timer
			if uint64(batchSize) < limit {
				// Process next batch immediately if there are items left
				if len(c.queue) > 0 {
					go c.processBatch() // Use a goroutine to avoid deadlock
				}
				return
			}
		}

		// Schedule the next batch processing if there are still items in the queue
		if len(c.queue) > 0 {
			// Ensure we wait for SERVICE_PERIOD before sending the next batch
			time.AfterFunc(period, func() {
				c.processBatch()
			})
		}
	} else {
		// If it's too soon to send the next batch, schedule it after the remaining time of the SERVICE_PERIOD
		remainingPeriod := period - timeSinceLastBatch
		time.AfterFunc(remainingPeriod, func() {
			c.processBatch()
		})
	}
}
