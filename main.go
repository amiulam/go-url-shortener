package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
)

// URLStore stores the mapping between short and long URLs
type URLStore struct {
	urls  map[string]string
	mutex sync.RWMutex
}

// Store instance
var store = &URLStore{
	urls: make(map[string]string),
}

// generateShortURL creates a random short URL
func generateShortURL() string {
	b := make([]byte, 4)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)[:6]
}

func main() {
	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Serve index.html
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			// Handle redirect for short URLs
			store.mutex.RLock()
			if longURL, exists := store.urls[r.URL.Path[1:]]; exists {
				store.mutex.RUnlock()
				http.Redirect(w, r, longURL, http.StatusFound)
				return
			}
			store.mutex.RUnlock()
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "index.html")
	})

	// Handle URL shortening
	http.HandleFunc("/shorten", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		longURL := r.FormValue("url")
		if longURL == "" {
			http.Error(w, "URL is required", http.StatusBadRequest)
			return
		}

		// Add http:// if no protocol is specified
		if !strings.HasPrefix(longURL, "http://") && !strings.HasPrefix(longURL, "https://") {
			longURL = "http://" + longURL
		}

		shortURL := generateShortURL()
		store.mutex.Lock()
		store.urls[shortURL] = longURL
		store.mutex.Unlock()

		// Return HTML fragment for HTMX
		tmpl := template.Must(template.New("result").Parse(`
			<div class="mt-4 p-4 bg-green-100 rounded-md">
				<p class="text-green-800 mb-2">URL Shortened Successfully!</p>
				<div class="flex items-center space-x-2">
					<input type="text" readonly value="{{.ShortURL}}" 
						class="flex-1 p-2 border rounded-md bg-white"
						id="shorturl-{{.ID}}">
					<button onclick="navigator.clipboard.writeText('{{.ShortURL}}')"
						class="px-4 py-2 bg-blue-500 text-white rounded-md hover:bg-blue-600">
						Copy
					</button>
				</div>
			</div>
		`))

		data := struct {
			ShortURL string
			ID       string
		}{
			ShortURL: fmt.Sprintf("http://%s/%s", r.Host, shortURL),
			ID:       shortURL,
		}

		tmpl.Execute(w, data)
	})

	fmt.Println("Server starting on :8000...")
	log.Fatal(http.ListenAndServe(":8000", nil))
}
