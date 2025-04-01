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
		} else if update.PreCheckoutQuery != nil {
			preCheckoutQueryHandler(bot, update) // Обрабатываем запрос на оплату
		} else if update.Message != nil && update.Message.SuccessfulPayment != nil {
			handleSuccessfulPayment(bot, update.Message) // Обрабатываем успешный платеж
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
		bot.Send(tgbotapi.NewMessage(chatID, "Привет! Используй /balance, /add_balance /buy или /withdraw_gift."))

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
	case "/buy":
		// Отправляем кнопку для Stars
		btn := tgbotapi.NewInlineKeyboardMarkup(
			tgbotapi.NewInlineKeyboardRow(
				tgbotapi.NewInlineKeyboardButtonData("Оплатить 1 ⭐", "pay_with_stars"),
			),
		)

		reply := tgbotapi.NewMessage(chatID, "Нажмите кнопку для оплаты 1 звездой:")
		reply.ReplyMarkup = btn
		bot.Send(reply)

		// ... остальные команды
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

func preCheckoutQueryHandler(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	preCheckoutQuery := update.PreCheckoutQuery
	if preCheckoutQuery != nil {
		preCheckoutConfig := tgbotapi.PreCheckoutConfig{
			PreCheckoutQueryID: preCheckoutQuery.ID,
			OK:                 true,
		}
		bot.Request(preCheckoutConfig)
	}
}

func handleSuccessfulPayment(bot *tgbotapi.BotAPI, message *tgbotapi.Message) {
	if message.SuccessfulPayment != nil {
		// Логика обработки успешного платежа
		log.Printf("Платеж на сумму %d подтвержден от пользователя %s", message.SuccessfulPayment.TotalAmount, message.From.UserName)

		// Допустим, начисляем баланс пользователя
		chatID := message.Chat.ID
		amount := message.SuccessfulPayment.TotalAmount / 100 // Переводим из центов в монеты
		addBalance(chatID, int(amount))

		// Подтверждение пользователю
		msg := tgbotapi.NewMessage(chatID, "Ваш платеж успешно получен. Спасибо за участие!")
		bot.Send(msg)
	}
}

func handleCallback(bot *tgbotapi.BotAPI, update tgbotapi.Update) {
	if update.CallbackQuery == nil {
		return
	}

	data := update.CallbackQuery.Data
	chatID := update.CallbackQuery.Message.Chat.ID

	if data == "pay_with_stars" {
		// Обработка оплаты Stars
		handleStarsPayment(bot, chatID)
	}

	bot.Request(tgbotapi.NewCallback(update.CallbackQuery.ID, ""))
}

func handleStarsPayment(bot *tgbotapi.BotAPI, chatID int64) {
	// 1. Начисляем бонус
	addBalance(chatID, 1) // 1 попытка за 1 звезду

	// 2. Уведомляем пользователя
	bot.Send(tgbotapi.NewMessage(chatID, "✅ Оплата прошла! Ваш баланс пополнен."))
}
