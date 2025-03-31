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

// Структуры данных
type Gift struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	Chance float64 `json:"chance"`
	Price  int32   `json:"price"`
}

type BalanceAdjustment struct {
	UserID string  `json:"user_id"`
	Delta  float64 `json:"delta"`
	Reason string  `json:"reason,omitempty"`
}

// Глобальные переменные
var (
	db      *sql.DB
	apiKeys = map[string]bool{
		os.Getenv("API_SECRET"): true,
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

// Инициализация БД
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

	// Создаем таблицу истории баланса, если ее нет
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS balance_history (
			id SERIAL PRIMARY KEY,
			user_id TEXT NOT NULL,
			delta NUMERIC NOT NULL,
			new_balance NUMERIC NOT NULL,
			reason TEXT,
			created_at TIMESTAMP DEFAULT NOW()
		)`)
	if err != nil {
		log.Fatal("Ошибка создания таблицы истории:", err)
	}

	fmt.Println("✅ База данных подключена и инициализирована")
}

// Middleware для проверки API ключа
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

// Получение баланса пользователя
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

// Обработчик получения баланса
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

// Изменение баланса (дельта)
func adjustBalanceHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var adj BalanceAdjustment
	if err := json.NewDecoder(r.Body).Decode(&adj); err != nil {
		http.Error(w, "Неверный формат запроса", http.StatusBadRequest)
		return
	}

	if adj.UserID == "" {
		http.Error(w, "user_id обязателен", http.StatusBadRequest)
		return
	}

	// Атомарное обновление баланса с проверкой
	var newBalance float64
	err := db.QueryRow(`
		UPDATE users 
		SET balance = balance + $1 
		WHERE telegram_id = $2 AND (balance + $1) >= 0
		RETURNING balance`,
		adj.Delta, adj.UserID).Scan(&newBalance)

	switch {
	case err == sql.ErrNoRows:
		http.Error(w, "Пользователь не найден или недостаточно средств", http.StatusBadRequest)
		return
	case err != nil:
		http.Error(w, "Ошибка базы данных", http.StatusInternalServerError)
		return
	}

	// Логируем изменение
	_, err = db.Exec(`
		INSERT INTO balance_history 
		(user_id, delta, new_balance, reason) 
		VALUES ($1, $2, $3, $4)`,
		adj.UserID, adj.Delta, newBalance, adj.Reason)
	if err != nil {
		log.Printf("Ошибка записи в историю: %v", err)
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"user_id":     adj.UserID,
		"new_balance": newBalance,
		"delta":       adj.Delta,
	})
}

// Получение случайного подарка
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

// Обработчик получения подарка
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
	// Проверка обязательных переменных окружения
	requiredEnv := []string{"API_SECRET", "DATABASE_URL"}
	for _, env := range requiredEnv {
		if os.Getenv(env) == "" {
			log.Fatalf("Требуется переменная окружения: %s", env)
		}
	}

	initDB()
	defer db.Close()

	// Регистрация обработчиков
	http.HandleFunc("/api/gift", apiKeyMiddleware(giftHandler))
	http.HandleFunc("/api/balance", apiKeyMiddleware(balanceHandler))
	http.HandleFunc("/api/adjust-balance", apiKeyMiddleware(adjustBalanceHandler))

	// Запуск сервера
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Сервер запущен на порту %s", port)
	log.Println("Доступные эндпоинты:")
	log.Println("GET  /api/gift - Получить случайный подарок")
	log.Println("GET  /api/balance?user_id=ID - Получить баланс")
	log.Println("POST /api/adjust-balance - Изменить баланс (delta)")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
