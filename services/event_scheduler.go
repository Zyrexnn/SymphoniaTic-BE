package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// MarkCompletedEventsOnce menandai konser yang telah melewati event_date_time + 6 jam sebagai COMPLETED
func MarkCompletedEventsOnce(ctx context.Context, db *sql.DB) (int64, error) {
	if db == nil {
		return 0, nil
	}
	res, err := db.ExecContext(ctx, `
		UPDATE events
		SET status = 'COMPLETED',
		    is_closed = TRUE,
		    status_updated_at = NOW()
		WHERE COALESCE(status, 'SCHEDULED') IN ('SCHEDULED', 'RESCHEDULED')
		  AND event_date_time IS NOT NULL
		  AND event_date_time + INTERVAL '6 hours' < NOW()
	`)
	if err != nil {
		return 0, err
	}
	affected, _ := res.RowsAffected()
	return affected, nil
}

// StartEventCompletionScheduler menjalankan ticker periodik untuk memperbarui status event selesai hingga ctx dibatalkan
func StartEventCompletionScheduler(ctx context.Context, db *sql.DB, interval time.Duration) {
	if interval <= 0 {
		interval = 15 * time.Minute
	}

	if updated, err := MarkCompletedEventsOnce(ctx, db); err != nil {
		log.Printf("[SCHEDULER-ERROR] Gagal memeriksa event selesai saat startup: %v\n", err)
	} else if updated > 0 {
		log.Printf("[SCHEDULER] %d konser otomatis ditandai COMPLETED\n", updated)
	}

	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if updated, err := MarkCompletedEventsOnce(ctx, db); err != nil {
					log.Printf("[SCHEDULER-ERROR] Gagal memperbarui status event COMPLETED: %v\n", err)
				} else if updated > 0 {
					log.Printf("[SCHEDULER] %d konser otomatis ditandai COMPLETED\n", updated)
				}
			}
		}
	}()
}
