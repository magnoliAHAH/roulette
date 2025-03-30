import React, { useState, useEffect } from 'react';
import axios from 'axios';
import './App.css';
import { IoAddCircleOutline } from "react-icons/io5";

function App() {
  const [isSpinning, setIsSpinning] = useState(false);
  const [gift, setGift] = useState(null);
  const [giftsList, setGiftsList] = useState([]);
  const [spinPosition, setSpinPosition] = useState(0);
  const [balance, setBalance] = useState(0)
  const [spinCost, setSpinCost] = useState(24)
  const [userId, setUserId] = useState(null);
  const addBalance = () => {
    setBalance(balance => balance + spinCost);
  };

  useEffect(() => {
    if (window.Telegram?.WebApp?.initDataUnsafe?.user) {
      const userId = window.Telegram.WebApp.initDataUnsafe.user.id;
      setUserId(userId);

      fetch(`https://supreme-roulette.work.gd/api/balance?user_id=${userId}`)
        .then((res) => res.json())
        .then((data) => setBalance(data.balance))
        .catch((err) => console.error("Ошибка запроса:", err));
    }
  }, []);

  const gifts = [
    { id: '1', name: 'Сердце', image: '/images/heart.png' },
    { id: '2', name: 'Мишка', image: '/images/bear.png' },
    { id: '3', name: 'Подарок', image: '/images/present.png' },
    { id: '4', name: 'Цветок', image: '/images/flower.png' },
    { id: '5', name: 'Торт', image: '/images/cake.png' },
    { id: '6', name: 'Букет', image: '/images/bouquet.png' },
    { id: '7', name: 'Кубок', image: '/images/cup.png' },
    { id: '8', name: 'Алмаз', image: '/images/diamond.png' }
  ];

  const startSpin = async () => {
    if (isSpinning) return;
    setBalance(balance - spinCost)
    setIsSpinning(true);
    setSpinPosition(0);

    try {
      const response = await axios.get(`https://supreme-roulette.work.gd/api/gift`);
            
      const selectedGift = response.data;
      setGift(selectedGift);

      const extendedGifts = generateGiftSequence(selectedGift);
      setGiftsList(extendedGifts);

      animateSpin(9);
    } catch (error) {
      console.error('Ошибка получения подарка:', error);
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

      {gift && !isSpinning && (
        <div>
          <h2>Вы выиграли: {gift.name}</h2>
          <img src={gift.image} alt={gift.name} />
          <div className='containerPriseBtn'>
            <button></button>
            <button></button>
          </div>
        </div>
      )}
    </div>
  );
}

export default App;

