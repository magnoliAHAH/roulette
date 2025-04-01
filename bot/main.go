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
	bot *tgbotapi.BotAPI
	db  *sql.DB
)

func main() {
	// Инициализация бота
	botToken := os.Getenv("TELEGRAM_BOT_TOKEN")
	var err error
	bot, err = tgbotapi.NewBotAPI(botToken)
	if err != nil {
		log.Fatal(err)
	}

	// Подключение к БД и создание таблиц
	initDB()
	defer db.Close()

	// Обработчик обновлений
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60
	updates := bot.GetUpdatesChan(u)

	log.Println("Бот запущен и ожидает сообщений...")

	for update := range updates {
		if update.Message != nil {
			handleMessage(update.Message)
		} else if update.PreCheckoutQuery != nil {
			handlePreCheckout(update.PreCheckoutQuery)
		} else if update.Message != nil && update.Message.SuccessfulPayment != nil {
			handleSuccessfulPayment(update.Message)
		} else if update.CallbackQuery != nil {
			handleCallback(update.CallbackQuery)
		}
	}
}

func initDB() {
	var err error
	db, err = sql.Open("postgres", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal(err)
	}

	// Создаем необходимые таблицы
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			telegram_id BIGINT UNIQUE NOT NULL,
			balance INT DEFAULT 0,
			created_at TIMESTAMP DEFAULT NOW()
		);
		
		CREATE TABLE IF NOT EXISTS payments (
			id SERIAL PRIMARY KEY,
			user_id BIGINT REFERENCES users(telegram_id),
			amount INT NOT NULL,
			payment_id TEXT NOT NULL UNIQUE,
			created_at TIMESTAMP DEFAULT NOW()
		);
	`)
	if err != nil {
		log.Fatal("Ошибка создания таблиц:", err)
	}
}

func handleMessage(msg *tgbotapi.Message) {
	switch {
	case msg.IsCommand() && msg.Command() == "start":
		handleStart(msg)
	case msg.IsCommand() && msg.Command() == "buy":
		showBuyOptions(msg.Chat.ID)
	case msg.Text == "Купить Stars":
		showBuyOptions(msg.Chat.ID)
	}
}

func handleStart(msg *tgbotapi.Message) {
	// Регистрируем пользователя, если его нет
	_, err := db.Exec(`
		INSERT INTO users (telegram_id) 
		VALUES ($1) 
		ON CONFLICT (telegram_id) DO NOTHING`,
		msg.From.ID,
	)
	if err != nil {
		log.Println("Ошибка регистрации пользователя:", err)
	}

	// Приветственное сообщение
	text := `👋 Добро пожаловать! 
Вы можете купить Stars для использования в боте.
Нажмите "Купить Stars" или используйте команду /buy`

	reply := tgbotapi.NewMessage(msg.Chat.ID, text)
	reply.ReplyMarkup = tgbotapi.NewReplyKeyboard(
		tgbotapi.NewKeyboardButtonRow(
			tgbotapi.NewKeyboardButton("Купить Stars"),
		),
	)
	bot.Send(reply)
}

func showBuyOptions(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "Выберите количество Stars:")
	msg.ReplyMarkup = tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("100 Stars (1$)", "buy:100"),
			tgbotapi.NewInlineKeyboardButtonData("500 Stars (5$)", "buy:500"),
		),
	)
	bot.Send(msg)
}

func handleCallback(query *tgbotapi.CallbackQuery) {
	if query.Data[:4] == "buy:" {
		starsAmount, err := strconv.Atoi(query.Data[4:])
		if err != nil {
			log.Println("Ошибка парсинга количества Stars:", err)
			return
		}

		// Отправляем инвойс
		if err := sendStarsInvoice(query.Message.Chat.ID, starsAmount); err != nil {
			log.Println("Ошибка отправки инвойса:", err)
			bot.Send(tgbotapi.NewMessage(query.Message.Chat.ID, "Ошибка создания платежа. Попробуйте позже."))
		}

		// Ответим на callback, чтобы убрать "часики" у кнопки
		callback := tgbotapi.NewCallback(query.ID, "")
		if _, err := bot.Request(callback); err != nil {
			log.Println("Ошибка ответа на callback:", err)
		}
	}
}

func sendStarsInvoice(chatID int64, starsAmount int) error {
	// Минимальная сумма - 1 звезда (100 единиц)
	if starsAmount < 1 {
		return fmt.Errorf("количество Stars должно быть положительным")
	}

	invoice := tgbotapi.NewInvoice(
		chatID,
		"Покупка Stars",
		"Пополнение баланса Telegram Stars",
		"stars_payload_"+strconv.FormatInt(chatID, 10),
		"",    // Пустой provider_token для Stars
		"",    // start_param можно оставить пустым
		"XTR", // Валюта Stars
		[]tgbotapi.LabeledPrice{
			{Label: "10 Stars", Amount: 1000}, // 100 Stars = 10000 единиц
		},
	)

	_, err := bot.Send(invoice)
	return err
}

func handlePreCheckout(query *tgbotapi.PreCheckoutQuery) {
	// В реальном приложении здесь можно добавить дополнительные проверки
	_, err := bot.Request(tgbotapi.PreCheckoutConfig{
		PreCheckoutQueryID: query.ID,
		OK:                 true,
	})
	if err != nil {
		log.Println("Ошибка подтверждения платежа:", err)
	}
}

func handleSuccessfulPayment(msg *tgbotapi.Message) {
	// Получаем данные о платеже
	payment := msg.SuccessfulPayment
	starsAmount := payment.TotalAmount / 100 // Конвертируем обратно в Stars

	// Сохраняем платеж в БД
	_, err := db.Exec(`
		INSERT INTO payments (user_id, amount, payment_id)
		VALUES ($1, $2, $3)`,
		msg.From.ID,
		starsAmount,
		payment.TelegramPaymentChargeID,
	)
	if err != nil {
		log.Println("Ошибка сохранения платежа:", err)
	}

	// Обновляем баланс пользователя
	_, err = db.Exec(`
		UPDATE users 
		SET balance = balance + $1 
		WHERE telegram_id = $2`,
		starsAmount,
		msg.From.ID,
	)
	if err != nil {
		log.Println("Ошибка обновления баланса:", err)
	}

	// Отправляем подтверждение пользователю
	reply := fmt.Sprintf("🎉 Спасибо за покупку! Ваш баланс пополнен на %d Stars.", starsAmount)
	bot.Send(tgbotapi.NewMessage(msg.Chat.ID, reply))
}
