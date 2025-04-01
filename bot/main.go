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
		sendStarsInvoice(bot, msg.Chat.ID) // Отправляем инвойс с кнопкой оплаты
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
func paymentKeyboard() tgbotapi.InlineKeyboardMarkup {
	button := tgbotapi.InlineKeyboardButton{
		Text: "Оплатить 1 ⭐️",
		Pay:  true,
	}
	keyboard := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(button),
	)
	return keyboard
}

func sendStarsInvoice(bot *tgbotapi.BotAPI, chatID int64) error {
	// 1. Настройка цен (1 звезда = 100 единиц)
	prices := []tgbotapi.LabeledPrice{
		{
			Label:  "Крутить рулетку",
			Amount: 100, // Минимальная сумма (1 звезда)
		},
	}

	// 2. Создаем конфигурацию инвойса
	invoice := tgbotapi.InvoiceConfig{
		BaseChat: tgbotapi.BaseChat{
			ChatID: chatID,
		},
		Title:               "Крутить рулетку (1 звезда)",
		Description:         "Платеж через Telegram Stars",
		Payload:             "unique_payload_" + strconv.FormatInt(chatID, 10),
		ProviderToken:       "", // Пусто для Stars
		Currency:            "XTR",
		Prices:              prices,
		SuggestedTipAmounts: []int{100}, // Чаевые (1 звезда)
		MaxTipAmount:        100000,     // Максимальные чаевые (1000 звезда)
		NeedName:            false,
		NeedPhoneNumber:     false,
		NeedEmail:           false,
		NeedShippingAddress: false,
		IsFlexible:          false,
	}

	// 3. Отправка инвойса
	_, err := bot.Send(invoice)
	if err != nil {
		return fmt.Errorf("ошибка отправки инвойса: %v", err)
	}
	return nil
}

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
