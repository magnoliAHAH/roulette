package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

type Gift struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	Chance float64 `json:"chance"`
	Price  int32   `json:"price"`
}

var (
	db      *sql.DB
	apiKeys = map[string]bool{
		os.Getenv("API_SECRET"): true, // Ключ берется из переменной окружения
	}
)

var gifts = []Gift{
	{"1", "Сердце", "/images/heart.png", 0.24, 15},
	{"2", "Мишка", "/images/bear.png", 0.24, 15},
	{"3", "Подарок", "/images/present.png", 0.14, 25},
	{"4", "Цветок", "/images/flower.png", 0.14, 25},
	{"5", "Торт", "/images/cake.png", 0.07, 50},
	{"6", "Букет", "/images/bouquet.png", 0.07, 50},
	{"7", "Кубок", "/images/cup.png", 0.035, 100},
	{"8", "Алмаз", "/images/diamond.png", 0.035, 100},
}

func initDB() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Ошибка подключения к БД:", err)
	}

	err = db.Ping()
	if err != nil {
		log.Fatal("База недоступна:", err)
	}

	fmt.Println("✅ База данных подключена")
}

func apiKeyMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-API-Key")
		if !apiKeys[apiKey] {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(w).Encode(map[string]string{"error": "Invalid API key"})
			return
		}
		next(w, r)
	}
}

func getUserBalance(userID string) (float64, error) {
	var balance float64
	err := db.QueryRow("SELECT balance FROM users WHERE telegram_id = $1", userID).Scan(&balance)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return balance, nil
}

func balanceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	userID := r.URL.Query().Get("user_id")
	if userID == "" {
		http.Error(w, "telegram_id is required", http.StatusBadRequest)
		return
	}

	balance, err := getUserBalance(userID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]float64{"balance": balance})
}

func getRandomGift() Gift {
	rand.Seed(time.Now().UnixNano())
	total := 0.0
	for _, g := range gifts {
		total += g.Chance
	}

	r := rand.Float64() * total
	cumulative := 0.0
	for _, g := range gifts {
		cumulative += g.Chance
		if r <= cumulative {
			return g
		}
	}
	return gifts[0]
}

func giftHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	json.NewEncoder(w).Encode(getRandomGift())
}

func main() {
	// Проверяем наличие API ключа
	if os.Getenv("API_SECRET") == "" {
		log.Fatal("API_SECRET environment variable must be set")
	}

	initDB()
	defer db.Close()

	// Защищенные маршруты
	http.HandleFunc("/api/gift", apiKeyMiddleware(giftHandler))
	http.HandleFunc("/api/balance", apiKeyMiddleware(balanceHandler))

	log.Println("Сервер запущен на http://localhost:8080")
	log.Println("Используйте заголовок X-API-Key с вашим секретным ключом")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
