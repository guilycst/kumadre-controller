package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"time"
)

var monitors = make(map[int64]map[string]interface{})
var nextID int64 = 1

func main() {
	http.HandleFunc("/", handleRoot)
	http.HandleFunc("/api/login", handleLogin)
	http.HandleFunc("/api/hostevents", handleHostEvents)
	http.HandleFunc("/api/monitor", handleMonitor)
	http.HandleFunc("/api/monitors", handleMonitors)
	http.HandleFunc("/api/debug/status-page", handleStatusPage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}

	fmt.Printf("Mock Uptime Kuma server starting on port %s\n", port)
	fmt.Println("Login with: admin/admin")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server error: %v\n", err)
	}
}

func handleRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok","version":"mock-1.0.0"}`))
}

func handleLogin(w http.ResponseWriter, r *http.Request) {
	var creds struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	json.NewDecoder(r.Body).Decode(&creds)

	if creds.Username == "admin" && creds.Password == "admin" {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"token":"mock-token-12345","userId":1}`))
		return
	}

	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"msg":"Invalid credentials"}`))
}

func handleHostEvents(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[]`))
}

func handleMonitor(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		var monitor map[string]interface{}
		json.NewDecoder(r.Body).Decode(&monitor)

		id := nextID
		nextID++
		monitor["id"] = id
		monitors[id] = monitor

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"msg":       "Monitor Created",
			"monitorId": id,
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`[]`))
}

func handleMonitors(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	monitorsList := make([]map[string]interface{}, 0)
	for _, m := range monitors {
		monitorsList = append(monitorsList, m)
	}

	json.NewEncoder(w).Encode(monitorsList)
}

func handleStatusPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	now := time.Now().Unix()
	rand.Seed(now)

	status := map[string]interface{}{
		"publicGroupList": []interface{}{
			map[string]interface{}{
				"id":   rand.Int63n(1000),
				"name": "Services",
				"monitorList": []interface{}{
					map[string]interface{}{
						"name":   "Test Service",
						"url":    "https://example.com",
						"status": rand.Intn(3),
						"uptime": 99.9,
					},
				},
			},
		},
	}

	json.NewEncoder(w).Encode(status)
}
