package main

import (
	"database/sql"
	"net/http"
	"time"
)

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
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func deleteTopicHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := getUserID(r)
		topicID := r.FormValue("topic_id")

		_, err := db.Exec(`
			DELETE FROM journals WHERE topic_id = ? AND topic_id IN (
				SELECT id FROM topics WHERE user_id = ?
			)`, topicID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_, err = db.Exec("DELETE FROM topics WHERE id = ? AND user_id = ?", topicID, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	})
}

func createJournalHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		topicID := r.FormValue("topic_id")
		content := r.FormValue("content")
		today := time.Now().In(appLocation).Format("2006-01-02")

		_, err := db.Exec(
			"INSERT INTO journals (topic_id, content, entry_date) VALUES (?, ?, ?)",
			topicID, content, today,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		http.Redirect(w, r, "/?slug="+r.FormValue("topic_slug"), http.StatusSeeOther)
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

func dashboardHandler(db *sql.DB) http.HandlerFunc {
	return requireAuth(func(w http.ResponseWriter, r *http.Request) {
		userID, _ := getUserID(r)
		today := time.Now().In(appLocation).Format("2006-01-02")

		rows, err := db.Query(`
			SELECT t.id, t.name, t.slug,
			       EXISTS(SELECT 1 FROM journals j WHERE j.topic_id = t.id AND j.entry_date = ?) as logged
			FROM topics t
			WHERE t.user_id = ?
			ORDER BY t.created_at DESC
		`, today, userID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var topics []TopicStatus
		for rows.Next() {
			var t TopicStatus
			rows.Scan(&t.ID, &t.Name, &t.Slug, &t.LoggedToday)
			topics = append(topics, t)
		}

		data := DashboardData{Topics: topics}

		slug := r.URL.Query().Get("slug")
		if slug != "" {
			for i := range topics {
				if topics[i].Slug == slug {
					data.SelectedTopic = &topics[i]
					break
				}
			}
			if data.SelectedTopic != nil {
				entryRows, _ := db.Query(
					"SELECT content, entry_date FROM journals WHERE topic_id = ? ORDER BY entry_date DESC",
					data.SelectedTopic.ID,
				)
				defer entryRows.Close()
				for entryRows.Next() {
					var e Entry
					entryRows.Scan(&e.Content, &e.Date)
					data.Entries = append(data.Entries, e)
				}
			}
		}

		renderTemplate(w, "dashboard.html", data)
	})
}
