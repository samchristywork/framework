package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
)

func generateRandomSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(b), nil
}

func sessionMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("session_id")
		if err != nil {
			sessionID, err := generateRandomSessionID()
			if err != nil {
				http.Error(w, "Failed to generate session ID", http.StatusInternalServerError)
				log.Printf("Error generating session ID: %v", err)
				return
			}

			cookie = &http.Cookie{
				Name:     "session_id",
				Value:    sessionID,
				Path:     "/",
				HttpOnly: true,
				Secure:   true,
			}

			http.SetCookie(w, cookie)
		}

		next.ServeHTTP(w, r)
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		firstFourOfCookie := "none"
		cookie, err := r.Cookie("session_id")
		if err == nil {
			firstFourOfCookie = cookie.Value[:4]
		}
		log.Printf("%s %s %s", r.Method, firstFourOfCookie, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

func middleware(next http.Handler) http.Handler {
	return sessionMiddleware(loggingMiddleware(next))
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", middleware(fs))
	log.Println("Server listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
