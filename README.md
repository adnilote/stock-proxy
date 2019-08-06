Сервис для получения стоимости акций различных компаний по HTTP REST API.

Источника данных - Alpha Vantage.
Документация: https://www.alphavantage.co/documentation/.

## Функции
Параметры https://www.alphavantage.co/documentation/

### Обязательные параметры:
- function
    стоимость за день, неделю, месяц.  
    function = {TIME_SERIES_INTRADAY, TIME_SERIES_DAILY, TIME_SERIES_DAILY_ADJUSTED, TIME_SERIES_WEEKLY, TIME_SERIES_WEEKLY_ADJUSTED, TIME_SERIES_MONTHLY, TIME_SERIES_MONTHLY_ADJUSTED}
- symbol
    тикер для бумаги (например, AMZN для Amazon)

### Дополнительные:
- interval
    для TIME_SERIES_INTRADAY пареметр interval обязателен
- outputsize
    outputsize = {compact, full}
- datatype
    datatype = {json, csv}

Пример:
- http://127.0.0.1:8082/sync/?function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn
- http://127.0.0.1:8082/async/?function=TIME_SERIES_DAILY&symbol=amzn


## Сервис имеет 3 метода:
- Получение стоимости акции (синхронно). Т.е. клиент ждет пока не получит ответ, хоть несколько часов (если сам не оборвет соединение).
    
    url: /sync/
    
    Пример: http://127.0.0.1:8082/sync/?function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn

- Получение стоимости акции (асинхронно). Т.е. клиент сразу получает сообщение о том, что его запрос принят (или не принят) и соединение обрывается.
    
    url: /async/
    
    Пример: http://127.0.0.1:8082/sync/?function=TIME_SERIES_INTRADAY&interval=1min&outputsize=compact&symbol=amzn

- Получение истории запросов по акции. История должна сохраняться после перезапуска сервиса или пересоздания контейнера.
    
    url: /history

    Пример: http://127.0.0.1:8082/history/?symbol=amzn

# Ограничения:
В бесплатной версии alphavantage есть ограничение 5 запросов в минуту.
Будем считать, что alphavontage банит нас при первом же превышении данного лимита.
Нужно реализовать это ограничение внутри нашего сервиса, чтобы он сам не позволял пользователям превышать лимит.

# Дополнение

- Сервис должен обладает методом /health для мониторинга состояния сервиса и зависимых компонентов
- Документация

    https://godoc.org/github.com/adnilote/stock-proxy/proxy
- Логирование
    * Все события в системе логируются
    * В зависимости от важности, события имеют соответствующий уровень логирования
    * Логи имеют структурированный формат - zap
    * Исключения и ошибки отправляются в Sentry
- Docker
    * Запуск тестов и компиляция происходят в отдельном контейнере.

# Запуск
Проект запускается при помощи команды docker-compose up