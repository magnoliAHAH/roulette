# Используем стабильный образ Node.js
FROM node:18
WORKDIR /app

# Копируем package.json и устанавливаем зависимости
COPY package*.json ./
RUN npm install

# Копируем весь код и собираем проект
COPY . .
RUN npm run build

# Устанавливаем serve для раздачи статических файлов
RUN npm install -g serve

# Экспортируем порт
EXPOSE 80

# Запускаем приложение, указывая верный путь к dist/
CMD ["serve", "-s", "dist", "-l", "80", "--single"]

