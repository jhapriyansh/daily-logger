package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

func checkMissingJournals(db *sql.DB, ntfyBase, ntfyUser, ntfyPass string) {
	today := time.Now().Format("2006-01-02")

	rows, err := db.Query(`
		SELECT t.name FROM topics t
		WHERE NOT EXISTS (
			SELECT 1 FROM journals j
			WHERE j.topic_id = t.id AND j.entry_date = ?
		)
	`, today)
	if err != nil {
		return
	}
	defer rows.Close()

	client := &http.Client{}
	url := fmt.Sprintf("%s/daily-logger", ntfyBase)

	for rows.Next() {
		var topicName string
		rows.Scan(&topicName)

		req, _ := http.NewRequest("POST", url, bytes.NewBufferString("Remember to log today"))
		req.Header.Set("Title", topicName)
		req.SetBasicAuth(ntfyUser, ntfyPass)
		client.Do(req)
	}
}

func startDailyCheck(db *sql.DB, ntfyBase, ntfyUser, ntfyPass string, checkHour int) {
	go func() {
		for {
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day(), checkHour, 0, 0, 0, now.Location())
			if now.After(next) {
				next = next.Add(24 * time.Hour)
			}
			time.Sleep(time.Until(next))
			checkMissingJournals(db, ntfyBase, ntfyUser, ntfyPass)
		}
	}()
}
