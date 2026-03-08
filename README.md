# Сервис расширения данных о персонах

### Описание

Этот сервис предназначен для хранения и обогащения информации о людях. Он позволяет создавать, получать, обновлять и удалять записи о людях, а также обогащать их данные возрастом, полом и национальностью с помощью внешних API.


### Установка и запуск с помощью Docker

```bash
git clone https://github.com/flexer2006/case-person-enrichment-go.git
cd case-person-enrichment-go
cp .env.example .env
cd deploy
docker-compose up -d
```

После запуска сервис доступен через Nginx по адресу `http://localhost:80`.
Внутри контейнера приложение слушает фиксированный порт **8080**; его не нужно маппировать вручную —
nginx проксирует трафик по сети Docker.

### Конфигурация

Основные настройки находятся в файле `deploy/.env`. Пример конфигурации можно найти в `.env.example`.

### Swagger UI

Интерактивная документация API доступна по адресу: `http://localhost/swagger/swagger.html`

### Базовый URL: `/api/v1`

| Метод | Путь                  | Описание                                      |
| ------ | --------------------- | ------------------------------------------------ |
| GET    | `/persons`            | Получить список персон с фильтрацией и постраничной загрузкой |
| GET    | `/persons/:id`        | Получить персону по ID                                 |
| POST   | `/persons`            | Создать новую персону                              |
| PUT    | `/persons/:id`        | Обновить персону                                  |
| PATCH  | `/persons/:id`        | Частично обновить персону                        |
| DELETE | `/persons/:id`        | Удалить персону                                  |
| POST   | `/persons/:id/enrich` | Обогатить данные персоны                               |

## Примеры использования API

### 1. Получение списка персон с фильтрацией

```bash
# Получить всех пользователей (первые 10).
curl -X GET "http://localhost/api/v1/persons"

# С применением фильтров.
curl -X GET "http://localhost/api/v1/persons?limit=5&offset=0&name=Ivan&gender=male"
```

Пример ответа:
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "name": "Ivan", 
      "surname": "Ivanov",
      "gender": "male",
      "gender_probability": 0.98,
      "nationality": "RU",
      "nationality_probability": 0.86
    }
  ],
  "total": 1,
  "limit": 5,
  "offset": 0
}
```

### 2. Создание новой персоны

```bash
curl -X POST "http://localhost/api/v1/persons" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Dmitry",
    "surname": "Ushakov",
    "patronymic": "Vasilievich"
  }'
```

### 3. Получение персоны по ID

```bash
curl -X GET "http://localhost/api/v1/persons/550e8400-e29b-41d4-a716-446655440001"
```

### 4. Обогащение данных персоны

```bash
curl -X POST "http://localhost/api/v1/persons/550e8400-e29b-41d4-a716-446655440001/enrich"
```

### 5. Обновление персоны

```bash
curl -X PUT "http://localhost/api/v1/persons/550e8400-e29b-41d4-a716-446655440001" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Dmitry",
    "surname": "Ushakov",
    "patronymic": "Alexeevich"
  }'
```

### 6. Удаление персоны

```bash
curl -X DELETE "http://localhost/api/v1/persons/550e8400-e29b-41d4-a716-446655440001"
```

## Внешние API для обогащения данных

Сервис использует следующие внешние API для обогащения данных:

- **Age**: [https://api.agify.io](https://api.agify.io) — определяет вероятный возраст по имени
- **Gender**: [https://api.genderize.io](https://api.genderize.io) — определяет вероятный пол по имени
- **Nationality**: [https://api.nationalize.io](https://api.nationalize.io) — определяет вероятную национальность по имени

## Структура базы данных

### Таблица `people`

| Поле | Тип | Описание |
|------|-----|----------|
| `id` | UUID | Первичный ключ, уникальный идентификатор персоны |
| `name` | VARCHAR(100) | Имя (обязательно) |
| `surname` | VARCHAR(100) | Фамилия (обязательно) |
| `patronymic` | VARCHAR(100) | Отчество (опционально) |
| `age` | INTEGER | Возраст |
| `gender` | VARCHAR(10) | Пол |
| `gender_probability` | DECIMAL(5,4) | Вероятность определения пола |
| `nationality` | VARCHAR(2) | Код страны (национальность) |
| `nationality_probability` | DECIMAL(5,4) | Вероятность определения национальности |
| `created_at` | TIMESTAMP WITH TIME ZONE | Дата и время создания записи |
| `updated_at` | TIMESTAMP WITH TIME ZONE | Дата и время последнего обновления записи |

## Миграции

Сервис автоматически применяет миграции при старте. Файлы миграций находятся в директории migrations.

## Скриншоты взаимодействия

Swagger:
![alt text](docs/photo/Swagger.png)

UI:
![alt text](docs/photo/Swagger-UI.png)

`http://localhost/api/v1/persons GET`:
![alt text](docs/photo/persons-get.png)

`http://localhost/api/v1/persons?limit=2&offset=0&name=Dmitry123`:
![alt text](docs/photo/persons-filters.png)

`http://localhost/api/v1/persons POST`:
![alt text](docs/photo/persons-post.png)

`http://localhost/api/v1/persons/{id} GET`:
![alt text](docs/photo/persons-id-get.png)

`http://localhost/api/v1/persons/550e8400-e29b-41d4-a716-446655440001/enrich POST`:
![alt text](docs/photo/persons-id-enrich-post.png)

`http://localhost/api/v1/persons/{id} PUT`:
![alt text](docs/photo/persons-id-put.png)

`http://localhost/api/v1/persons/{id} PATCH`:
![alt text](docs/photo/persons-id-patch.png)

`http://localhost/api/v1/persons/{id} DELETE`:
![alt text](docs/photo/persons-id-delete.png)