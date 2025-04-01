package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"

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
		if withdrawGift(chatID, 200) {
			bot.Send(tgbotapi.NewMessage(chatID, "🎁 Вы успешно вывели подарок!"))
		} else {
			bot.Send(tgbotapi.NewMessage(chatID, "❌ Недостаточно монет для вывода подарка."))
		}
	case "/buy":
		sendStarsInvoice(chatID)
	case "Поддержать проект":
		bot.Send(tgbotapi.NewMessage(chatID, "Введите количество ⭐ для оплаты:"))
	default:
		if _, err := strconv.Atoi(msg.Text); err == nil {
			processStarsAmount(msg)
		}
	}
}

func sendStarsInvoice(chatID int64) {
	prices := []tgbotapi.LabeledPrice{
		{Label: "Крутить рулетку", Amount: 100}, // 1 звезда = 100 единиц
	}

	invoice := tgbotapi.InvoiceConfig{
		BaseChat:            tgbotapi.BaseChat{ChatID: chatID},
		Title:               "Крутить рулетку (1 звезда)",
		Description:         "Платеж через Telegram Stars",
		Payload:             "spin_" + strconv.FormatInt(chatID, 10),
		ProviderToken:       "", // Пусто для Stars
		Currency:            "XTR",
		Prices:              prices,
		SuggestedTipAmounts: []int{100}, // Чаевые (1 звезда)
		MaxTipAmount:        100,        // Максимальные чаевые
	}

	if _, err := bot.Send(invoice); err != nil {
		log.Printf("Ошибка отправки инвойса: %v", err)
	}
}

func processStarsAmount(msg *tgbotapi.Message) {
	starsAmount, _ := strconv.Atoi(msg.Text)
	if starsAmount <= 0 {
		bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "Введите положительное число"))
		return
	}

	prices := []tgbotapi.LabeledPrice{
		{Label: "Поддержка проекта", Amount: starsAmount * 100},
	}

	invoice := tgbotapi.InvoiceConfig{
		BaseChat:            tgbotapi.BaseChat{ChatID: msg.Chat.ID},
		Title:               fmt.Sprintf("Поддержка проекта (%d ⭐)", starsAmount),
		Description:         "Спасибо за вашу поддержку!",
		Payload:             "donation_" + strconv.FormatInt(msg.Chat.ID, 10),
		ProviderToken:       "",
		Currency:            "XTR",
		Prices:              prices,
		SuggestedTipAmounts: []int{starsAmount * 100},
		MaxTipAmount:        starsAmount * 100,
	}

	if _, err := bot.Send(invoice); err != nil {
		log.Printf("Ошибка отправки инвойса: %v", err)
	}
}

func handleCallback(query *tgbotapi.CallbackQuery) {
	chatID := query.Message.Chat.ID

	switch query.Data {
	case "pay_with_stars":
		// Обработка оплаты Stars
		_, err := bot.Send(tgbotapi.NewMessage(chatID, "Готово! Нажмите на кнопку оплаты в появившемся сообщении."))
		if err != nil {
			log.Println("Ошибка отправки сообщения:", err)
		}
		sendStarsInvoice(chatID)

		// Другие callback-действия можно добавить здесь
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
	starsReceived := msg.SuccessfulPayment.TotalAmount / 100
	addBalance(msg.Chat.ID, starsReceived)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, fmt.Sprintf("✅ Получено %d звёзд! Спасибо за поддержку!", starsReceived)))
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
