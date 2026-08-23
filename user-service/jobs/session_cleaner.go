package jobs

import (
	"context"
	"database/sql"
	"log"
	"sync"
	"time"
)

const deleteExpiredSessions = `
	DELETE FROM sessions
	WHERE expires_at < now()`

func StartSessionCleaner(db *sql.DB, interval time.Duration) (stop func()) {
	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-done:
				return
			case <-ticker.C:
				ctx, cancel := context.WithTimeout(context.Background(), interval)
				if _, err := db.ExecContext(ctx, deleteExpiredSessions); err != nil {
					log.Printf("session cleanup: %v", err)
				}
				cancel()
			}
		}
	}()

	return func() {
		close(done)
		wg.Wait()
	}
}
