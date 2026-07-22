package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

var appLocation *time.Location = time.UTC

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	ntfyBase := os.Getenv("NTFY_BASE")
	ntfyUser := os.Getenv("NTFY_USER")
	ntfyPass := os.Getenv("NTFY_PASS")

	if ntfyBase == "" || ntfyUser == "" || ntfyPass == "" {
		log.Fatal("NTFY_BASE, NTFY_USER, and NTFY_PASS must be set")
	}

	checkHour := 21
	if h := os.Getenv("CHECK_HOUR"); h != "" {
		parsed, err := strconv.Atoi(h)
		if err != nil {
			log.Fatalf("invalid CHECK_HOUR: %v", err)
		}
		checkHour = parsed
	}

	tzName := os.Getenv("CHECK_TIMEZONE")
	if tzName == "" {
		tzName = "UTC"
	}
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		log.Fatalf("invalid CHECK_TIMEZONE %q: %v", tzName, err)
	}
	appLocation = loc

	startDailyCheck(db, ntfyBase, ntfyUser, ntfyPass, checkHour, loc)

	http.HandleFunc("/", dashboardHandler(db))
	http.HandleFunc("/signup", signupHandler(db))
	http.HandleFunc("/login", loginHandler(db))
	http.HandleFunc("/logout", logoutHandler)
	http.HandleFunc("/topics/create", createTopicHandler(db))
	http.HandleFunc("/topics/delete", deleteTopicHandler(db))
	http.HandleFunc("/journals/create", createJournalHandler(db))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
