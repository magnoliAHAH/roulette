package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

var db *sql.DB

func main() {
	// Загружаем переменные из .env

	// Подключаемся к PostgreSQL
	initDB()
	defer db.Close()

	// Создаём таблицу, если её нет
	createTable()

	// Создаём бота
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	bot, err := tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal("Ошибка при создании бота:", err)
	}

	// Запуск слушателя сообщений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	fmt.Println("🤖 Бот запущен!")

	for update := range updates {
		if update.Message != nil {
			handleMessage(bot, update.Message)
		}
	}
}

// 🔹 Подключение к БД
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

// 🔹 Создание таблицы
func createTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		telegram_id BIGINT UNIQUE NOT NULL,
		balance INT DEFAULT 100
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Ошибка создания таблицы:", err)
	}
	fmt.Println("✅ Таблица users создана")
}

// 🔹 Обработка сообщений
func handleMessage(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	addUser(chatID) // Регистрируем пользователя в БД

	switch msg.Text {
	case "/start":
		bot.Send(tgbotapi.NewMessage(chatID, "Привет! Используй /balance, /add_balance или /withdraw_gift."))

	case "/balance":
		balance := getBalance(chatID)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("💰 Ваш баланс: %d монет.", balance)))

	case "/add_balance":
		addBalance(chatID, 50)
		balance := getBalance(chatID)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Баланс пополнен! Новый баланс: %d монет.", balance)))

	case "/withdraw_gift":
		if withdrawGift(chatID, 200) { // Стоимость подарка 200 монет
			bot.Send(tgbotapi.NewMessage(chatID, "🎁 Вы успешно вывели подарок!"))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Недостаточно монет для вывода подарка."))
		}
	}
}

// 🔹 Добавление пользователя
func addUser(telegramID int64) {
	_, err := db.Exec("INSERT INTO users (telegram_id) VALUES ($1) ON CONFLICT DO NOTHING", telegramID)
	if err != nil {
		log.Println("Ошибка при добавлении пользователя:", err)
	}
}

// 🔹 Получение баланса
func getBalance(telegramID int64) int {
	var balance int
	err := db.QueryRow("SELECT balance FROM users WHERE telegram_id = $1", telegramID).Scan(&balance)
	if err != nil {
		log.Println("Ошибка при получении баланса:", err)
		return 0
	}
	return balance
}

// 🔹 Пополнение баланса
func addBalance(telegramID int64, amount int) {
	_, err := db.Exec("UPDATE users SET balance = balance + $1 WHERE telegram_id = $2", amount, telegramID)
	if err != nil {
		log.Println("Ошибка при обновлении баланса:", err)
	}
}

// 🔹 Вывод подарка (уменьшение баланса)
func withdrawGift(telegramID int64, cost int) bool {
	balance := getBalance(telegramID)
	if balance < cost {
		return false
	}

	_, err := db.Exec("UPDATE users SET balance = balance - $1 WHERE telegram_id = $2", cost, telegramID)
	if err != nil {
		log.Println("Ошибка при снятии баланса:", err)
		return false
	}
	return true
}

// обработка платежей
