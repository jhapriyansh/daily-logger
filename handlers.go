package main

import (
	"database/sql"
	"net/http"
	"os"
	"time"
)

func appLocation() *time.Location {
	name := os.Getenv("APP_TIMEZONE")
	if name == "" {
		name = "Asia/Kolkata"
	}

	location, err := time.LoadLocation(name)
	if err != nil {
		return time.Local
	}
	return location
}

func today() string {
	return time.Now().In(appLocation()).Format("2006-01-02")
}

func loginHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderTemplate(w, "login.html", nil)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")

		var id int
		var hash string
		err := db.QueryRow("SELECT id, password_hash FROM users WHERE username = ?", username).Scan(&id, &hash)
		if err != nil || !checkPassword(password, hash) {
			http.Error(w, "invalid credentials", http.StatusUnauthorized)
			return
		}
		createSession(w, id)
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func createTopicHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := getUserID(r)
		name := r.FormValue("name")
		slug := slugify(name)
		_, err := db.Exec("INSERT INTO topics (user_id, name, slug) VALUES (?, ?, ?)", userID, name, slug)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/topic?slug="+slug, http.StatusSeeOther)
	})
}

func deleteTopicHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		userID, _ := getUserID(r)
		topicID := r.FormValue("topic_id")

		tx, err := db.Begin()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var ownedTopicID int
		err = tx.QueryRow("SELECT id FROM topics WHERE id = ? AND user_id = ?", topicID, userID).Scan(&ownedTopicID)
		if err == sql.ErrNoRows {
			http.Error(w, "topic not found", http.StatusNotFound)
			return
		}
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if _, err = tx.Exec("DELETE FROM journals WHERE topic_id = ?", ownedTopicID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if _, err = tx.Exec("DELETE FROM topics WHERE id = ?", ownedTopicID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err = tx.Commit(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func createJournalHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := getUserID(r)
		topicID := r.FormValue("topic_id")
		content := r.FormValue("content")

		var slug string
		err := db.QueryRow("SELECT slug FROM topics WHERE id = ? AND user_id = ?", topicID, userID).Scan(&slug)
		if err != nil {
			http.Error(w, "topic not found", http.StatusNotFound)
			return
		}

		_, err = db.Exec(
			"INSERT INTO journals (topic_id, content, entry_date) VALUES (?, ?, ?)",
			topicID, content, today(),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/topic?slug="+slug, http.StatusSeeOther)
	})
}

func signupHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			renderTemplate(w, "signup.html", nil)
			return
		}
		username := r.FormValue("username")
		password := r.FormValue("password")

		hash, err := hashPassword(password)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		_, err = db.Exec("INSERT INTO users (username, password_hash) VALUES (?, ?)", username, hash)
		if err != nil {
			http.Error(w, "username taken", http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

type TopicStatus struct {
	ID          int
	Name        string
	Slug        string
	LoggedToday bool
}

type Entry struct {
	Content string
	Date    string
}

type DashboardData struct {
	Topics        []TopicStatus
	SelectedTopic *TopicStatus
	Entries       []Entry
}

func loadTopics(db *sql.DB, userID int) ([]TopicStatus, error) {
	rows, err := db.Query(`
		SELECT t.id, t.name, t.slug,
		       EXISTS(SELECT 1 FROM journals j WHERE j.topic_id = t.id AND j.entry_date = ?) as logged
		FROM topics t
		WHERE t.user_id = ?
		ORDER BY t.created_at DESC
	`, today(), userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var topics []TopicStatus
	for rows.Next() {
		var topic TopicStatus
		if err := rows.Scan(&topic.ID, &topic.Name, &topic.Slug, &topic.LoggedToday); err != nil {
			return nil, err
		}
		topics = append(topics, topic)
	}
	return topics, rows.Err()
}

func renderDashboard(w http.ResponseWriter, r *http.Request, db *sql.DB, userID int, slug string) {
	topics, err := loadTopics(db, userID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := DashboardData{Topics: topics}
	if len(topics) == 0 {
		renderTemplate(w, "dashboard.html", data)
		return
	}

	selectedIndex := 0
	if slug != "" {
		selectedIndex = -1
		for i := range topics {
			if topics[i].Slug == slug {
				selectedIndex = i
				break
			}
		}
		if selectedIndex == -1 {
			http.NotFound(w, r)
			return
		}
	}

	selected := topics[selectedIndex]
	data.SelectedTopic = &selected
	rows, err := db.Query(`
		SELECT content, strftime('%Y-%m-%d', entry_date)
		FROM journals
		WHERE topic_id = ?
		ORDER BY entry_date DESC, id DESC
	`, selected.ID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var entry Entry
		if err := rows.Scan(&entry.Content, &entry.Date); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.Entries = append(data.Entries, entry)
	}
	if err := rows.Err(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	renderTemplate(w, "dashboard.html", data)
}

func dashboardHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := getUserID(r)
		renderDashboard(w, r, db, userID, "")
	})
}

func topicViewHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := getUserID(r)
		slug := r.URL.Query().Get("slug")
		renderDashboard(w, r, db, userID, slug)
	})
}
