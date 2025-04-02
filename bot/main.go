package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	_ "github.com/lib/pq"
)

var db *sql.DB
var userStates = make(map[int64]string)

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
		log.Panic(err)
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message != nil {
			chatID := update.Message.Chat.ID
			addUser(chatID)

			if state, ok := userStates[chatID]; ok && state == "waiting_amount" {
				amount, err := strconv.Atoi(update.Message.Text)
				if err != nil || amount <= 0 {
					bot.Send(tgbotapi.NewMessage(chatID, "❌ Введите положительное число"))
					continue
				}
				sendInvoice(bot, chatID, amount)
				delete(userStates, chatID)
				continue
			}

			switch update.Message.Text {
			case "/start":
				bot.Send(tgbotapi.NewMessage(chatID, "Привет! Используй /buy для покупки звёзд."))
			case "/buy":
				sendBuyMenu(bot, chatID)
			}

			// Обработка успешного платежа
			if update.Message.SuccessfulPayment != nil {
				handleSuccessfulPayment(bot, update.Message)
			}
		}

		if update.CallbackQuery != nil {
			chatID := update.CallbackQuery.Message.Chat.ID
			data := update.CallbackQuery.Data

			if strings.HasPrefix(data, "buy_stars:") {
				amount, _ := strconv.Atoi(strings.Split(data, ":")[1])
				sendInvoice(bot, chatID, amount)
			} else if data == "custom_amount" {
				msg := tgbotapi.NewMessage(chatID, "Введите количество звёзд:")
				bot.Send(msg)
				userStates[chatID] = "waiting_amount"
			}

			// Важно: подтверждаем callback
			callback := tgbotapi.NewCallback(update.CallbackQuery.ID, "")
			if _, err := bot.Request(callback); err != nil {
				log.Println("Callback error:", err)
			}
		}

		if update.PreCheckoutQuery != nil {
			// Важно: правильно подтверждаем preCheckout
			_, err := bot.Request(tgbotapi.PreCheckoutConfig{
				PreCheckoutQueryID: update.PreCheckoutQuery.ID,
				OK:                 true,
			})
			if err != nil {
				log.Println("PreCheckout error:", err)
			}
		}
	}
}

func sendBuyMenu(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "🎯 Выберите количество звёзд:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("24 ⭐", "buy_stars:24"),
			tgbotapi.NewInlineKeyboardButtonData("50 ⭐", "buy_stars:50"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("100 ⭐", "buy_stars:100"),
			tgbotapi.NewInlineKeyboardButtonData("Другое количество", "custom_amount"),
		),
	)
	bot.Send(msg)
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
	case "/buy":
		sendStarsSelection(bot, chatID) // Отправляем инвойс для оплаты
	}
}

func sendStarsSelection(bot *tgbotapi.BotAPI, chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "⭐ Выберите количество звёзд:")
	markup := tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("24 ⭐", "buy_stars:24"),
			tgbotapi.NewInlineKeyboardButtonData("50 ⭐", "buy_stars:50"),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("100 ⭐", "buy_stars:100"),
			tgbotapi.NewInlineKeyboardButtonData("Другое количество", "custom_amount"),
		),
	)
	msg.ReplyMarkup = markup
	bot.Send(msg)
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
func sendInvoice(bot *tgbotapi.BotAPI, chatID int64, amount int) {

	invoiceConfig := tgbotapi.InvoiceConfig{
		BaseChat:            tgbotapi.BaseChat{ChatID: chatID},
		Title:               "title",
		Description:         "description",
		Payload:             "{}",
		ProviderToken:       "", // Для «Звёзд» оставляем пустым
		Currency:            "XTR",
		Prices:              []tgbotapi.LabeledPrice{{Label: "Diamond", Amount: amount}},
		SuggestedTipAmounts: []int{},
	}

	_, err := bot.Send(invoiceConfig)
	if err != nil {
		log.Printf("Ошибка при отправке инвойса: %v", err)
	}
}
func handleSuccessfulPayment(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
	chatID := msg.Chat.ID
	payload := msg.SuccessfulPayment.InvoicePayload

	// Извлекаем количество из payload
	parts := strings.Split(payload, ":")
	if len(parts) != 2 {
		log.Println("Неверный формат payload:", payload)
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка обработки платежа (неверный формат)"))
		return
	}

	amount, err := strconv.Atoi(parts[1])
	if err != nil {
		log.Println("Ошибка парсинга количества:", err)
		bot.Send(tgbotapi.NewMessage(chatID, "❌ Ошибка обработки платежа"))
		return
	}

	// Зачисляем средства
	addBalance(chatID, amount)

	// Отправляем подтверждение
	balance := getBalance(chatID)
	response := fmt.Sprintf("✅ Платеж успешен! Зачислено %d звёзд.\nВаш баланс: %d", amount, balance)
	bot.Send(tgbotapi.NewMessage(chatID, response))
}
