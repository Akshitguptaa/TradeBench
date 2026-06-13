package orchestrator

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tradebench/bots/internal/bot"
	"github.com/tradebench/bots/internal/consumer"
)

// Run spins up a goroutine pool for the requested duration and target RPS.
// It manages the bots and forwards their telemetry events.
func Run(ctx context.Context, event consumer.RunStartedEvent, telemetryCh chan<- consumer.TelemetryEvent) {
	log.Printf("bots orchestrator: starting run_id=%s, target_rps=%d, duration=%ds",
		event.RunID, event.TargetRPS, event.DurationSecs)

	// bot per 10 RPS, min 1 bot.
	workerCount := event.TargetRPS / 10
	if workerCount < 1 {
		workerCount = 1
	}

	// Calculate how many orders each worker needs to send per second to hit TargetRPS.
	rpsPerWorker := float64(event.TargetRPS) / float64(workerCount)
	delayBetweenOrders := time.Duration(float64(time.Second) / rpsPerWorker)

	endTime := time.Now().Add(time.Duration(event.DurationSecs) * time.Second)

	var wg sync.WaitGroup
	wg.Add(workerCount)

	for i := 0; i < workerCount; i++ {
		botID := fmt.Sprintf("bot-%s-%d", event.RunID[:8], i)
		workerBot := bot.New(botID)

		go func(b *bot.Bot, idx int) {
			defer wg.Done()

			// Stagger worker start times to prevent thundering herd
			staggerDelay := time.Duration(idx) * (delayBetweenOrders / time.Duration(workerCount))
			time.Sleep(staggerDelay)

			ticker := time.NewTicker(delayBetweenOrders)
			defer ticker.Stop()

			for {
				select {
				case <-ctx.Done():
					return
				case t := <-ticker.C:
					if t.After(endTime) {
						return
					}
					// SendOrder generates the payload, hits the sandbox, and returns telemetry
					telem := b.SendOrder(ctx, event.RunID, event.SandboxAddress)
					select {
					case telemetryCh <- telem:
					case <-ctx.Done():
						return
					}
				}
			}
		}(workerBot, i)
	}

	wg.Wait()
	log.Printf("bots orchestrator: finished run_id=%s", event.RunID)
}
