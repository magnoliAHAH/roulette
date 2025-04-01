import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';
import { IoAddCircleOutline } from "react-icons/io5";

function App() {
  const API_KEY = "dev_5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"
  const [isSpinning, setIsSpinning] = useState(false);
  const [gift, setGift] = useState(null);
  const [giftsList, setGiftsList] = useState([]);
  const [spinPosition, setSpinPosition] = useState(0);
  const [balance, setBalance] = useState(0)
  const [spinCost, setSpinCost] = useState(24)
  const [userId, setUserId] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const addBalance = () => {
    setBalance(balance => balance + spinCost);
  };

  useEffect(() => {
    if (window.Telegram?.WebApp?.initDataUnsafe?.user) {
      const userId = window.Telegram.WebApp.initDataUnsafe.user.id;
      setUserId(userId);

      fetch(`https://supreme-roulette.work.gd/api/balance?user_id=${userId}`, 
        {
          headers: {
            'X-API-Key': API_KEY,
          }
        }
      )
        .then((res) => res.json())
        .then((data) => setBalance(data.balance))
        .catch((err) => console.error("Ошибка запроса:", err));
    }
  }, []);

  const gifts = [
    { id: '1', name: 'Сердце', image: '/images/heart.png', price: '15' },
    { id: '2', name: 'Мишка', image: '/images/bear.png', price: '15' },
    { id: '3', name: 'Подарок', image: '/images/present.png', price: '25' },
    { id: '4', name: 'Цветок', image: '/images/flower.png', price: '25' },
    { id: '5', name: 'Торт', image: '/images/cake.png', price: '50' },
    { id: '6', name: 'Букет', image: '/images/bouquet.png', price: '50' },
    { id: '7', name: 'Кубок', image: '/images/cup.png', price: '100' },
    { id: '8', name: 'Алмаз', image: '/images/diamond.png', price: '100' }
  ];
  const closeModal = () => {
    setShowModal(false);
  };

  const sellButtonHandler = () => {
    setBalance(balance + gift.price)
    setShowModal(false);
  }

  const startSpin = async () => {
    if (isSpinning || balance < spinCost) return;
  
    const prevBalance = balance; // Для отката при ошибке
    setIsSpinning(true);
    setShowModal(false);
  
    try {
      // 1. Отправляем запрос на списание
      const adjustResponse = await axios.post(
        'https://supreme-roulette.work.gd/api/adjust-balance',
        {
          user_id: userId, // Важно: используем динамический ID!
          delta: -spinCost, // Списываем полную стоимость
          reason: "Roulette spin"
        },
        {
          headers: { 
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
          },
          timeout: 10000
        }
      );
  
      // 2. Проверяем ответ сервера
      if (!adjustResponse.data?.success) {
        throw new Error('Сервер не подтвердил списание');
      }
  
      // 3. Обновляем баланс на фронтенде
      setBalance(adjustResponse.data.new_balance);
  
      // 4. Запускаем рулетку
      const giftResponse = await axios.get(
        'https://supreme-roulette.work.gd/api/gift',
        { headers: { 'X-API-Key': API_KEY } }
      );
      
      setGift(giftResponse.data);
      setGiftsList(generateGiftSequence(giftResponse.data));
      animateSpin(9);
  
    } catch (error) {
      console.error('Ошибка:', error);
      setBalance(prevBalance); // Откат баланса
      setIsSpinning(false);
    }
  };

  const generateGiftSequence = (winningGift) => {
    const randomGiftCount = Math.floor(Math.random() * 60) + 25;
    let shuffledGifts = shuffleArray([...gifts]);

    const extendedGifts = [];
    for (let i = 0; i < randomGiftCount; i++) {
      extendedGifts.push(shuffledGifts[i % shuffledGifts.length]);
    }
    extendedGifts[20] = winningGift;
    return extendedGifts;
  };

  const animateSpin = (winningIndex) => {
    let position = 0;
    const duration = 3000; // Длительность вращения
    const steps = duration / 20; // Количество шагов
    let step = 0;
  
    const spinInterval = setInterval(() => {
      step++;
      const easingFactor = 1 - step / steps;
      position += 20 * easingFactor; // Увеличиваем позицию для вращения
      setSpinPosition(position);
      
      // Проверяем, достигли ли мы конца анимации
      if (step >= steps) {
        clearInterval(spinInterval);
        setIsSpinning(false);
        setShowModal(true);
      }
    }, 30);
  };

  const shuffleArray = (array) => {
    let shuffledArray = [...array];
    for (let i = shuffledArray.length - 1; i > 0; i--) {
      const j = Math.floor(Math.random() * (i + 1));
      [shuffledArray[i], shuffledArray[j]] = [shuffledArray[j], shuffledArray[i]];
    }
    return shuffledArray;
  };

  return (
    <div className="app">
      
      <div className='upper-menu'>
        <div>
          <h1>{userId}</h1>
        </div>

        <div className='converted-starts'>
          <img src='/images/stars-logo.png' className='stars-image'></img>
          <div>{balance}</div>
          <IoAddCircleOutline size={"30px"} onClick={addBalance}/>
        </div>
      </div>

      <div className="wheel-container">
        <div className="wheel" style={{ transform: `translateX(-${spinPosition}px)` }}>
          {giftsList.map((gift, index) => (
            <div key={index} className="gift-item">
              <img src={gift.image} alt={gift.name} />
              <div>{gift.name}</div>
              <div className="gift-label">Элемент {index + 1}</div> {/* Подпись для каждого элемента */}
            </div>
          ))}
        </div>
        <div className="pointer">|</div>
      </div>

      
      <button onClick={startSpin} disabled={isSpinning || balance < spinCost} className='spinBtn'>{balance < spinCost ? "Недостаточный баланс" : "Крутить рулетку"}</button>

      {showModal && gift && (
        <div className="modal-overlay">
          <div className="modal">
            <h2>Поздравляем!</h2>
            <p>Вы выиграли: {gift.name}</p>
            <img src={gift.image} alt={gift.name} />
            <div>
              <button onClick={sellButtonHandler} className="modal-close-btn">
                Продать
              </button>
              <button onClick={closeModal} className="modal-close-btn">
                Вывести
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;
