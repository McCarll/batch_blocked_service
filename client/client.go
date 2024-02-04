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
	if uint64(currentQueueSize) >= c.queueLimit {
		go c.processBatch()
	} else if c.processTimer == nil {
		go c.processQueue()
	}
}

func (c *Client) processQueue() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if uint64(len(c.queue)) < c.queueLimit && c.processTimer == nil {
		_, p := c.service.GetLimits()
		c.processTimer = time.AfterFunc(p, func() {
			c.mu.Lock()
			c.processTimer = nil
			c.mu.Unlock()
			c.processBatch()
		})
	}
}

func (c *Client) startProcessingTimer() {
	_, p := c.service.GetLimits()
	c.processTimer = time.AfterFunc(p, func() {
		c.processQueue()
	})
}

func (c *Client) processBatch() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()

	limit, period := c.service.GetLimits()

	timeSinceLastBatch := now.Sub(c.lastBatchTime)

	batchSize := int(limit)
	if len(c.queue) < batchSize {
		batchSize = len(c.queue)
	}

	if batchSize == 0 {
		return
	}

	if timeSinceLastBatch >= period || len(c.queue) < int(limit) {
		batch := make(service.Batch, 0, batchSize)
		batch, c.queue = c.queue[:batchSize], c.queue[batchSize:]

		c.mu.Unlock()
		err := c.service.Process(context.Background(), batch)
		c.mu.Lock()

		if err != nil {
			log.Printf("Error processing batch: %v", err)
			if err == service.ErrBlocked {
				// If the service is blocked, requeue the batch and don't update the lastBatchTime
				c.queue = append(batch, c.queue...)
			}
		} else {
			log.Printf("Processed a batch of %d items successfully.", batchSize)
			c.lastBatchTime = now

			if uint64(batchSize) < limit {
				if len(c.queue) > 0 {
					go c.processBatch()
				}
				return
			}
		}

		if len(c.queue) > 0 {
			time.AfterFunc(period, func() {
				c.processBatch()
			})
		}
	} else {
		remainingPeriod := period - timeSinceLastBatch
		time.AfterFunc(remainingPeriod, func() {
			c.processBatch()
		})
	}
}
