# Этап сборки
FROM node:18-alpine as build

# Устанавливаем рабочую директорию внутри контейнера
WORKDIR /app

# Копируем файлы зависимостей
COPY package*.json ./

# Устанавливаем зависимости
RUN npm install

# Копируем остальные исходники в контейнер
COPY . .

# Выполняем сборку фронтенда (если нужно)
RUN npm run build

# Этап запуска
FROM node:18-alpine

# Устанавливаем рабочую директорию для приложения
WORKDIR /app

# Копируем файлы из этапа сборки
COPY --from=build /app .

# Открываем порт для Node.js приложения
EXPOSE 3000

# Запускаем сервер Node.js (предположим, что в package.json есть скрипт "start")
CMD ["npm", "start"]
