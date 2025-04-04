package main

import (
	"bytes"
	"compress/gzip"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// Структуры данных
type Gift struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Image     string  `json:"image"`
	Chance    float64 `json:"chance"`
	Price     int32   `json:"price"`
	FileId    string  `json:"fileId"`
	LocalPath string  `json:"localPath"`
}

type BalanceAdjustment struct {
	UserID string  `json:"user_id"`
	Delta  float64 `json:"delta"`
	Reason string  `json:"reason,omitempty"`
}

// Структура для ответа Telegram API
type TelegramFileResponse struct {
	OK     bool `json:"ok"`
	Result struct {
		FilePath string `json:"file_path"`
	} `json:"result"`
}

type LottieResponse struct {
	FilePath string `json:"file_path"`
	Content  string `json:"content,omitempty"`
}

// Глобальные переменные
var (
	db      *sql.DB
	apiKeys = map[string]bool{
		os.Getenv("API_SECRET"): true,
	}
)

var gifts = []Gift{
	{"5170145012310081615", "Сердце", "stickers/file_0.tgs", 0.24, 15, "CAACAgIAAxUAAWftiPMLxu8EQ_qjkagqWLE-MaYRAAJ8GwACkoTYS_itxFzPIpm1NgQ", "/images/heart-Photoroom.png"},
	{"5170233102089322756", "Мишка", "stickers/file_1.tgs", 0.24, 15, "CAACAgIAAxUAAWfuP2WNsmgwExdYGcr5cI1Ahyz7AAKtVAACmj_pSnvPkDQv_sivNgQ", "/images/bear-Photoroom.png"},
	{"5170250947678437525", "Подарок", "stickers/file_2.tgs", 0.14, 25, "CAACAgIAAxUAAWfuRg15KexG01ACPLcWEX5ODZLcAAKNSAACsU84SMuB4ge7V_wGNgQ", "/images/present-Photoroom.png"},
	{"5168103777563050263", "Цветок", "stickers/file_3.tgs", 0.14, 25, "CAACAgIAAxUAAWfudcO964qSuz51KReUK7z5EMI8AAJxGQAC1P-BS9CYmaAv74KENgQ", "/images/flower-Photoroom.png"},
	{"5170144170496491616", "Торт", "stickers/file_4.tgs", 0.07, 50, "CAACAgIAAxUAAWfuVQfRaYk5xGetaiVgU7jfqVZxAAKBGAAClZ-JSmrGT5kBZFDGNgQ", "/images/cake-Photoroom.png"},
	{"5170314324215857265", "Букет", "stickers/file_5.tgs", 0.07, 50, "CAACAgIAAxUAAWfudD3UkW3jRGrhCO09_nDonZRyAALeJQACRSDgSsjmE6AUz0NeNgQ", "/images/bouquet-Photoroom.png"},
	{"5168043875654172773", "Кубок", "stickers/file_6.tgs", 0.035, 100, "CAACAgIAAxUAAWfudcOLzWLnanjwucTziB55VplxAAKZGwACzagQS38R6izkF899NgQ", "/images/cup-Photoroom.png"},
	{"5170521118301225164", "Алмаз", "stickers/file_7.tgs", 0.035, 100, "CAACAgIAAxUAAWfudcPM1jN3IVNejHx2YBZcgq5OAAIbHgACQEjwS6XKX-OK2k1MNgQ", "/images/diamond-Photoroom.png"},
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

// Обработчик для получения file_path
func filePathHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "file_id is required", http.StatusBadRequest)
		return
	}

	// Формируем URL запроса к Telegram API
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	if botToken == "" {
		http.Error(w, "TELEGRAM_BOT_TOKEN not configured", http.StatusInternalServerError)
		return
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)

	// Выполняем запрос
	resp, err := http.Get(url)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error requesting Telegram API: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Декодируем ответ
	var fileResp TelegramFileResponse
	if err := json.NewDecoder(resp.Body).Decode(&fileResp); err != nil {
		http.Error(w, fmt.Sprintf("Error decoding response: %v", err), http.StatusInternalServerError)
		return
	}

	if !fileResp.OK {
		http.Error(w, "Telegram API returned not OK", http.StatusInternalServerError)
		return
	}

	// Возвращаем только file_path
	json.NewEncoder(w).Encode(map[string]string{
		"file_path": fileResp.Result.FilePath,
	})
}

func lottieHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-API-Key")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	fileID := r.URL.Query().Get("file_id")
	if fileID == "" {
		http.Error(w, "file_id is required", http.StatusBadRequest)
		return
	}

	// 1. Получаем путь к файлу
	filePath, err := getTelegramFilePath(fileID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error getting file path: %v", err), http.StatusInternalServerError)
		return
	}

	// 2. Получаем содержимое файла (опционально)
	content := ""
	if r.URL.Query().Get("with_content") == "true" {
		fileContent, err := getLottieFileContent(filePath)
		if err != nil {
			http.Error(w, fmt.Sprintf("Error getting file content: %v", err), http.StatusInternalServerError)
			return
		}
		content = string(fileContent)
	}

	json.NewEncoder(w).Encode(LottieResponse{
		FilePath: filePath,
		Content:  content,
	})
}

func getTelegramFilePath(fileID string) (string, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/bot%s/getFile?file_id=%s", botToken, fileID)

	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var result struct {
		OK     bool `json:"ok"`
		Result struct {
			FilePath string `json:"file_path"`
		} `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if !result.OK {
		return "", fmt.Errorf("telegram API returned not OK")
	}

	return result.Result.FilePath, nil
}

func getLottieFileContent(filePath string) ([]byte, error) {
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	url := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", botToken, filePath)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	compressedData, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Распаковываем .tgs файл (Zlib compressed)
	reader, err := gzip.NewReader(bytes.NewReader(compressedData))
	if err != nil {
		return nil, err
	}
	defer reader.Close()

	return io.ReadAll(reader)
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
	http.HandleFunc("/api/lottie", apiKeyMiddleware(lottieHandler))

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
	log.Println("GET  /api/lottie?file_id=ID - Получить Lottie-файл")
	log.Println("GET  /api/lottie?file_id=ID&with_content=true - Получить Lottie-файл с содержимым")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}
