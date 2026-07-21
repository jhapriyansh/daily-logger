package main

import (
	"log"
	"net/http"
	"os"
)

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

	startDailyCheck(db, ntfyBase, ntfyUser, ntfyPass, 21)

	http.HandleFunc("/", dashboardHandler(db))
	http.HandleFunc("/signup", signupHandler(db))
	http.HandleFunc("/login", loginHandler(db))
	http.HandleFunc("/topics/create", createTopicHandler(db))
	http.HandleFunc("/journals/create", createJournalHandler(db))
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.HandleFunc("/topic", topicViewHandler(db))

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
