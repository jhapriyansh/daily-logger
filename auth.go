package main

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
	
	"golang.org/x/crypto/bcrypt"
)
var sessions = map[string]int{}

func hashPassword(pw string) (string, error){
	bytes, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	return string(bytes), err
}

func checkPassword(pw, hash string) (bool){
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

func generateToken() string {
	b := make([]byte, 32)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func createSession(w http.ResponseWriter, userID int){
	token := generateToken()
	sessions[token] = userID
	http.SetCookie(w, &http.Cookie{
		Name : "session",
		Value : token,
		HttpOnly : true,
		Expires : time.Now().Add(30*24*time.Hour),
		Path : "/",
	})
}

func getUserID(r *http.Request) (int, bool) {
	cookie, err := r.Cookie("session")
	if err != nil {
		return 0, false
	}
	userID, ok := sessions[cookie.Value]
	return userID, ok
}

func requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if _, ok := getUserID(r); !ok {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}


