package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

var (
	db  *sql.DB
	bot *tgbotapi.BotAPI
)

func main() {
	// Подключаемся к PostgreSQL
	initDB()
	defer db.Close()

	// Создаём таблицу, если её нет
	createTable()

	// Создаём бота
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
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
			handleMessage(update.Message)
		} else if update.CallbackQuery != nil {
			handleCallback(update.CallbackQuery)
		} else if update.PreCheckoutQuery != nil {
			handlePreCheckout(update.PreCheckoutQuery)
		} else if update.Message != nil && update.Message.SuccessfulPayment != nil {
			handleSuccessfulPayment(update.Message)
		}
	}
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

func handleMessage(msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	addUser(chatID)

	switch {
	case msg.IsCommand() && msg.Command() == "start":
		bot.Send(tgbotapi.NewMessage(chatID, "Привет! Используй /balance, /add_balance /buy или /withdraw_gift."))
	case msg.IsCommand() && msg.Command() == "balance":
		balance := getBalance(chatID)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("💰 Ваш баланс: %d монет.", balance)))
	case msg.IsCommand() && msg.Command() == "add_balance":
		addBalance(chatID, 50)
		balance := getBalance(chatID)
		bot.Send(tgbotapi.NewMessage(chatID, fmt.Sprintf("✅ Баланс пополнен! Новый баланс: %d монет.", balance)))
	case msg.IsCommand() && msg.Command() == "withdraw_gift":
		if withdrawGift(chatID, 200) {
			bot.Send(tgbotapi.NewMessage(chatID, "🎁 Вы успешно вывели подарок!"))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Недостаточно монет для вывода подарка."))
		}
	case msg.IsCommand() && msg.Command() == "buy":
		sendStarsRequest(chatID)
	case msg.Text == "Поддержать проект":
		sendStarsRequest(chatID)
	default:
		if starsAmount, err := strconv.Atoi(msg.Text); err == nil {
			processStarsAmount(msg.Chat.ID, starsAmount)
		}
	}
}

func sendStarsRequest(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Введите количество ⭐ для оплаты:")
	bot.Send(msg)
}

func processStarsAmount(chatID int64, starsAmount int) {
	if starsAmount <= 0 {
		bot.Send(tgbotapi.NewMessage(chatID, "Пожалуйста, введите положительное число."))
		return
	}

	// Создаем инвойс для Stars
	prices := []tgbotapi.LabeledPrice{
		{Label: "Поддержка проекта", Amount: starsAmount * 100}, // 1 звезда = 100 единиц
	}

	invoice := tgbotapi.InvoiceConfig{
		BaseChat:      tgbotapi.BaseChat{ChatID: chatID},
		Title:         fmt.Sprintf("Поддержка проекта (%d ⭐)", starsAmount),
		Description:   "Спасибо за вашу поддержку!",
		Payload:       "donation_" + strconv.FormatInt(chatID, 10) + "_" + strconv.FormatInt(time.Now().Unix(), 10),
		ProviderToken: "",
		Currency:      "XTR",
		Prices:        prices,
	}

	// Добавляем кнопку оплаты
	btn := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("Оплатить %d ⭐", starsAmount),
				"pay_stars",
			),
		),
	)
	invoice.ReplyMarkup = btn

	if _, err := bot.Send(invoice); err != nil {
		log.Printf("Ошибка отправки инвойса: %v", err)
		bot.Send(tgbotapi.NewMessage(chatID, "Ошибка при создании платежа"))
	}
}

func handleCallback(query *tgbotapi.CallbackQuery) {
	if query.Data == "pay_stars" {
		sendStarsRequest(query.Message.Chat.ID)
	}

	// Ответ на callback-запрос
	callbackCfg := tgbotapi.CallbackConfig{
		CallbackQueryID: query.ID,
	}
	if _, err := bot.Request(callbackCfg); err != nil {
		log.Println("Ошибка ответа на callback:", err)
	}
}

func handlePreCheckout(query *tgbotapi.PreCheckoutQuery) {
	_, err := bot.Request(tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: query.ID,
		OK:                 true,
	})
	if err != nil {
		log.Println("PreCheckout error:", err)
	}
}

func handleSuccessfulPayment(msg *tgbotapi.Message) {
	if msg.SuccessfulPayment == nil {
		return
	}

	starsReceived := msg.SuccessfulPayment.TotalAmount / 100
	addBalance(msg.Chat.ID, starsReceived)

	reply := fmt.Sprintf("✅ Получено %d звёзд! Спасибо за поддержку!", starsReceived)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, reply))
}

// Функции работы с БД
func addUser(telegramID int64) {
	_, err := db.Exec("INSERT INTO users (telegram_id) VALUES ($1) ON CONFLICT DO NOTHING", telegramID)
	if err != nil {
		log.Println("Ошибка при добавлении пользователя:", err)
	}
}

func getBalance(telegramID int64) int {
	var balance int
	err := db.QueryRow("SELECT balance FROM users WHERE telegram_id = $1", telegramID).Scan(&balance)
	if err != nil {
		log.Println("Ошибка при получении баланса:", err)
		return 0
	}
	return balance
}

func addBalance(telegramID int64, amount int) {
	_, err := db.Exec("UPDATE users SET balance = balance + $1 WHERE telegram_id = $2", amount, telegramID)
	if err != nil {
		log.Println("Ошибка при обновлении баланса:", err)
	}
}

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
