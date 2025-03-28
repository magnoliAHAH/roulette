import React, { useState } from 'react';
import axios from 'axios';
import './App.css';

function App() {
  const [isSpinning, setIsSpinning] = useState(false);
  const [gift, setGift] = useState(null);
  const [giftsList, setGiftsList] = useState([]);
  const [spinPosition, setSpinPosition] = useState(0);

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

    setIsSpinning(true);
    setSpinPosition(0);

    try {
      const response = await axios.get('http://localhost:8080/api/gift');
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
    
    const stopPosition = Math.floor(Math.random() * extendedGifts.length);
    extendedGifts[21] = winningGift;
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


      <button onClick={startSpin} disabled={isSpinning}>Крутить рулетку</button>

      {gift && !isSpinning && (
        <div>
          <h2>Вы выиграли: {gift.name}</h2>
          <img src={gift.image} alt={gift.name} />
        </div>
      )}
    </div>
  );
}

export default App;

