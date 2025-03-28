# Этап сборки приложения
FROM node:18-alpine as build

WORKDIR /app

# Копируем package.json и package-lock.json для установки зависимостей
COPY package*.json ./

# Устанавливаем зависимости
RUN npm install

# Копируем весь исходный код в контейнер
COPY . .

# Запускаем сборку фронтенда
RUN npm run build

# Этап сервера с Nginx
FROM nginx:alpine

# Копируем собранные файлы фронтенда в папку, где Nginx ожидает статичные файлы
COPY --from=build /app/dist /usr/share/nginx/html

# Открываем порты для Nginx
EXPOSE 80

# Запускаем Nginx
CMD ["nginx", "-g", "daemon off;"]
