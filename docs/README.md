[![golangci-lint](https://github.com/KSerditov/Trading/actions/workflows/golang-lint.yml/badge.svg)](https://github.com/KSerditov/Trading/actions/workflows/golang-lint.yml)

## Описание
### Биржа
Отдельное приложение, подгружает данные тикеров из текстовых файлов (игнорируя дату и отрезая тикеры, время которых было до старта), затем скармливает их каждую секунду сама себе.
При получении нового тикера занимается обработкой заявок в стакане и сбором агреггированных данных по тикерам.
Рассылает агреггированные данные.
Интерфейс для клиентов - gRPC.

### Брокер
Подключается к бирже по gRPC, предоставляет http api для клиентов.
При получении заявки от клиента, перенаправляет её на биржу, сохраняя данные у себя в mysql.
При получении ответа или агреггированных данных от биржи так же сохраняет их.

### Клиент
Встроенные в брокер web страницы на bootstrap, получающие данные от брокера

![image](https://user-images.githubusercontent.com/3009597/211049871-6c831121-41f3-42dc-b4b2-118f65cb531c.png)
![image](https://user-images.githubusercontent.com/3009597/211049933-96d9784b-3216-455f-a624-194af42eac5c.png)
![image](https://user-images.githubusercontent.com/3009597/211050005-fc2dd493-4b1e-41f6-adf6-96e9da43741f.png)

### Телеграм бот с Oauth авторизацией через Vk
![image](https://user-images.githubusercontent.com/3009597/212405672-27e77649-dc6d-4482-bc56-316cdca6df35.png)

### Тестовый пользователь
megaurich
12345678

## Известные проблемы/недоделки
### Биржа
1. Добавить grpc аутентификацию
2. Добавить логгирование
3. Проверять уникальность соединений брокеров
4. Покрыть тестами торговлю
5. Добавить конфигурационные параметры
6. Сохранять состояние биржи в базе и продолжать без потери данных при перезапуске
7. Проверить и привести к единому виду типы данных тикеров и сделок
8. Сделать докер образ

### Брокер
1. Дописать аннотации для Swagger (изначальный код был сгенерен из openapi описание, но далеко оттуда ушел)
2. Включить Swagger UI в docker compose и сделать на него ссылку
3. Добавить тесты
4. Добавить редирект на логин в случае истекшего времени действия токена
5. Нужны проверки на переполнение volume, price, balance при торговле
6. Сохранять выполненные сделки в order_history
7. Автоматическое переподсоединение стримов Statistics/Results после обрыва.
8. Придумать способ получать пропущенные данные после обрыва (нужна поддержка на стороне биржи)
9. Сделать докер образ

### Клиент
1. Логирование

### Общее
1. Починить линтер в гитхабе
2. Добавить мониторинг

### Запуск
1. > make start-docker
2. > go run ./cmd/exchange/main.go
3. > go run ./cmd/broker/main.go
4. > go run ./cmd/tgclient/main.go
5. http://localhost:8080/ OR https://t.me/GoLangCourse2023Bot

