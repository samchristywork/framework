package main

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"os"
	"strings"
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
				Name:   "session_id",
				Value:  sessionID,
				Path:   "/",
				Secure: true,
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

func listFiles(dir string) []string {
	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatalf("Failed to read directory %s: %v", dir, err)
	}
	var fileList []string
	for _, file := range files {
		if !file.IsDir() {
			fileList = append(fileList, file.Name())
		}
	}
	return fileList
}

func handleFiles(staticFiles map[string]string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileName := r.URL.Path[1:]

		if fileName == "" {
			fileName = "index.html"
		}

		content, exists := staticFiles[fileName]
		if !exists {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(content))
	}
}

func transclude(fileName string) string {
	content, err := os.ReadFile("./static/" + fileName)
	log.Printf("Transcluding file: %s", fileName)
	if err != nil {
		log.Fatalf("Failed to read file %s: %v", fileName, err)
	}

	s := ""
	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if len(line) != 0 && line[0] == '!' {
			filename := line[1:]
			s += transclude(filename)
		} else {
			s += line + "\n"
		}
	}

	return s
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	files := listFiles("./static")

	staticFiles := make(map[string]string)
	for _, file := range files {
		staticFiles[file] = transclude(file)
	}

	http.Handle("/", middleware(handleFiles(staticFiles)))
	log.Println("Server listening on port " + port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
