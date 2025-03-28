package main

import (
	"encoding/json"
	"math/rand"
	"net/http"
	"time"
)

type Gift struct {
	ID     string  `json:"id"`
	Name   string  `json:"name"`
	Image  string  `json:"image"`
	Chance float64 `json:"chance"`
}

var gifts = []Gift{
	{"1", "Сердце", "/images/heart.png", 0.413},    // 41.3% шанс на подарок
	{"2", "Мишка", "/images/bear.png", 0.248},      // 24.8% шанс на подарок
	{"3", "Подарок", "/images/present.png", 0.124}, // 12.4% шанс на подарок
	{"4", "Цветок", "/images/flower.png", 0.0413},  // 4.13% шанс на подарок
	{"5", "Торт", "/images/cake.png", 0.0413},      // 4.13% шанс на подарок
	{"6", "Букет", "/images/bouquet.png", 0.0413},  // 4.13% шанс на подарок
	{"7", "Кубок", "/images/cup.png", 0.0413},      // 4.13% шанс на подарок
	{"8", "Алмаз", "/images/diamond.png", 0.0413},  // 4.13% шанс на подарок
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
	return gifts[0] // fallback
}

func giftHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")             // Разрешить запросы с любых источников
	w.Header().Set("Access-Control-Allow-Methods", "GET")          // Разрешить методы
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type") // Разрешить заголовки
	if r.Method == http.MethodOptions {
		// Для предзапроса (OPTIONS) просто возвращаем 200
		w.WriteHeader(http.StatusOK)
		return
	}

	json.NewEncoder(w).Encode(getRandomGift())
}

func main() {
	http.HandleFunc("/api/gift", giftHandler)
	http.ListenAndServe(":8080", nil)
}
