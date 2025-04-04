import React, { useState, useEffect } from 'react';
import axios from 'axios';
import Lottie from "lottie-react"
import './App.css';
import { IoAddCircleOutline } from "react-icons/io5";

function App() {
  const API_KEY = "dev_5e884898da28047151d0e56f8dc6292773603d0d6aabbdd62a11ef721d1542d8"
  const BOT_TOKEN = "7513080511:AAFQHYyrZROaysopau2WF3Qi8NjtTj7p0q4"
  const [isSpinning, setIsSpinning] = useState(false);
  const [gift, setGift] = useState(null);
  const [giftsList, setGiftsList] = useState([]);
  const [spinPosition, setSpinPosition] = useState(0);
  const [balance, setBalance] = useState(0)
  const [spinCost, setSpinCost] = useState(24)
  const [userId, setUserId] = useState(null);
  const [showModal, setShowModal] = useState(false);
  const [isSending, setIsSending] = useState(false);
  const [stickerData, setStickerData] = useState(null);
  const [notification, setNotification] = useState(null);
  const [firstName, setFirstName] = useState('');
  const [userAvatar, setUserAvatar] = useState('');

  useEffect(() => {
    if (notification) {
      const timer = setTimeout(() => {
        setNotification(null);
      }, 2500);
      return () => clearTimeout(timer);
    }
  }, [notification]);

  const sendGift = async (userId, giftId) => {
    if (!userId || !giftId) {
      throw new Error('Не указан пользователь или подарок');
    }

    const url = `https://api.telegram.org/bot${BOT_TOKEN}/sendGift`;
  
    const params = {
      user_id: userId,
      gift_id: giftId,
      pay_for_upgrade: false,
    };
  
    try {
      const response = await axios.post(url, params);
      if (response.data.ok) {
        console.log('Подарок успешно отправлен:', response.data.result);
        setNotification({ type: 'success', message: 'Подарок успешно отправлен!' });
      } else {
        console.error('Ошибка при отправке подарка:', response.data.description);
        setNotification({ type: 'error', message: `Ошибка: ${response.data.description}` });
      }
    } catch (error) {
      console.error('Ошибка при отправке подарка:', error.message);
      setNotification({ type: 'error', message: `Ошибка сети: ${error.message}` });
    }
    setShowModal(false)
  };

  const addBalance = () => {
    setNotification({ type: 'error', message: `Пополни баланс через бота /buy` });
    
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
    { id: '1', name: 'Сердце', image: '/images/heart-Photoroom.png', price: '15' },
    { id: '2', name: 'Мишка', image: '/images/bear-Photoroom.png', price: '15' },
    { id: '3', name: 'Подарок', image: '/images/present-Photoroom.png', price: '25' },
    { id: '4', name: 'Цветок', image: '/images/flower-Photoroom.png', price: '25' },
    { id: '5', name: 'Торт', image: '/images/cake-Photoroom.png', price: '50' },
    { id: '6', name: 'Букет', image: '/images/bouquet-Photoroom.png', price: '50' },
    { id: '7', name: 'Кубок', image: '/images/cup-Photoroom.png', price: '100' },
    { id: '8', name: 'Алмаз', image: '/images/diamond-Photoroom.png', price: '100' }
  ];
  const closeModal = () => {
    setShowModal(false);
  };

  const sellButtonHandler = async () => {
    const prevBalance = balance;
    try {
      // 1. Списание баланса (точно как в работающем cURL)
      const adjustResponse = await axios.post(
        'https://supreme-roulette.work.gd/api/adjust-balance',
        {
          user_id: String(userId),
          delta: gift.price,
          reason: "Selled Gift"
        },
        {
          headers: { 
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
          },
          timeout: 10000
        }
      );
  
      // 2. Валидация ответа
      if (!adjustResponse.data?.success) {
        throw new Error('Balance adjustment failed: ' + JSON.stringify(adjustResponse.data));
      }
  
      // 3. Логирование для отладки
      console.log('Balance adjusted:', adjustResponse.data);
  
      // 4. Обновляем баланс на фронтенде
      setBalance(prevBalance + gift.price); 
      setNotification({ type: 'success', message: 'Подарок успешно продан!' });
    } catch (error) {
      console.error('Error in Selling:', error);
      setNotification({ type: 'error', message: 'Ошибка продажи' });
      
      setBalance(prevBalance); // Откат
      
      // Дополнительная диагностика
      if (error.response) {
        console.error('Server response:', error.response.data);
        setNotification({ type: 'error', message: `Ошибка ответа: ${error.response.data}` });
      }
    }
    setShowModal(false);
  }

  const startSpin = async () => {
    if (isSpinning || balance < spinCost) return;
  
    const prevBalance = balance;
    setIsSpinning(true);
    setShowModal(false);
    setStickerData(null); // Сбрасываем предыдущие данные стикера
  
    try {
      // 1. Списание баланса
      const adjustResponse = await axios.post(
        'https://supreme-roulette.work.gd/api/adjust-balance',
        {
          user_id: String(userId),
          delta: -spinCost,
          reason: "Spin"
        },
        {
          headers: { 
            'X-API-Key': API_KEY,
            'Content-Type': 'application/json'
          },
          timeout: 10000
        }
      );
  
      if (!adjustResponse.data?.success) {
        throw new Error('Balance adjustment failed: ' + JSON.stringify(adjustResponse.data));
      }
  
      setBalance(prevBalance - spinCost);
  
      // 2. Получаем подарок
      const giftResponse = await axios.get(
        'https://supreme-roulette.work.gd/api/gift',
        { 
          headers: { 'X-API-Key': API_KEY },
          timeout: 10000
        }
      );
      
      const receivedGift = giftResponse.data;
      setGift(receivedGift);
      setGiftsList(generateGiftSequence(receivedGift));
      animateSpin(9);
      // 3. Получаем данные стикера
      if (receivedGift.fileId) {
        const stickerResponse = await axios.get(
          `https://supreme-roulette.work.gd/api/lottie?file_id=${receivedGift.fileId}&with_content=true`,
          { 
            headers: { 'X-API-Key': API_KEY },
            timeout: 10000
          }
        );
        
        // Проверяем и обрабатываем ответ
        if (stickerResponse.data?.content) {
          try {
            const animationData = JSON.parse(stickerResponse.data.content);
            setStickerData(animationData);
          } catch (parseError) {
            console.error('Error parsing sticker content:', parseError);
            setNotification({ type: 'error', message: `Ошибка парсинга: ${parseError}` });
            // Если не удалось распарсить, сохраняем как есть
            setStickerData(stickerResponse.data);
          }
        } else {
          // Если структура ответа отличается
          setStickerData(stickerResponse.data);
          setNotification({ type: 'error', message: `Структура ответа отличается` });
        }
      }
  
      
  
    } catch (error) {
      console.error('Error in startSpin:', error);
      setBalance(prevBalance);
      setIsSpinning(false);
      setNotification({ type: 'error', message: `Ошибка прокрута: ${error}` });
      
      if (error.response) {
        console.error('Server response:', error.response.data);
        setNotification({ type: 'error', message: `Ошибка ответа: ${error.response.data}` });
      }
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
              
              <img src={gift.image} alt={gift.name}/>
            
                
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
            <p>Стоимостью {gift.price}</p>
            <Lottie animationData={stickerData} loop={true}/>
            <div>
              <button onClick={sellButtonHandler} className="modal-close-btn">
                Продать
              </button>
              <button 
        onClick={async () => {
          if (!userId || !gift?.id) return;
          
          setIsSending(true);
          try {
            await sendGift(userId, gift.id);
            setShowModal(false);
          } catch (error) {
            console.error(error);
          } finally {
            setIsSending(false);
          }
        }}
        disabled={isSending}
        className="modal-close-btn"
      >
        {isSending ? "Отправка..." : "Вывести"}
      </button>
            </div>
          </div>
        </div>
      )}

      {notification && (
        <div className={`notification ${notification.type}`}>
          {notification.message}
          <button onClick={() => setNotification(null)} className="notification-close">
            ×
          </button>
        </div>
      )}
    </div>
  );
}

export default App;