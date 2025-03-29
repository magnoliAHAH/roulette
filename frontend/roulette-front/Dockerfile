FROM node:18-alpine

WORKDIR /app

# Копируем package.json и package-lock.json
COPY package*.json ./

# Устанавливаем зависимости
RUN npm install

# Копируем все файлы
COPY . .

# Собираем фронтенд (если используешь React/Vite)
RUN npm run build

# Запускаем сервер
EXPOSE 3000
CMD ["npm", "start"]