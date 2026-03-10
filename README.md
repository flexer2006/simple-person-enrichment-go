# Person Enrichment Service

API для хранения и обогащения данных людей (возраст, пол, национальность).

## Запуск

```sh
git clone https://github.com/flexer2006/pes-api.git
cd simple-person-enrichment-go
cp .env.example .env
cd deploy && docker-compose up -d
```

Доступ: `http://localhost`. Swagger UI: `http://localhost/swagger/swagger.html`.

## API (base `/api/v1`)

- `GET  /persons` – список (фильтры: name, surname, patronymic, gender, nationality, age; limit/offset)
- `GET  /persons/:id` – получить
- `POST /persons` – создать
- `PUT|PATCH /persons/:id` – обновить
- `DELETE /persons/:id` – удалить
- `POST /persons/:id/enrich` – добавить возраст/пол/национальность

### Примеры использования API

1. Получение списка персон

```bash
curl -X GET "http://localhost/api/v1/persons"
```

![alt text](docs/photo/persons-get.png)

1.1. С фильтрами

```bash
curl -X GET "http://localhost/api/v1/persons?limit=5&offset=0&name=Ivan&gender=male"
```

![alt text](docs/photo/persons-filters.png)

2. Создание

```bash
curl -X POST "http://localhost/api/v1/persons" \
  -H "Content-Type: application/json" \
  -d '{"name":"Dmitry","surname":"Ushakov","patronymic":"Vasilievich"}'
```

![alt text](docs/photo/persons-post.png)

3. Получение по ID

```bash
curl -X GET "http://localhost/api/v1/persons/{UID}"
```

![alt text](docs/photo/persons-id-get.png)

4. Обогащение

```bash
curl -X POST "http://localhost/api/v1/persons/{UID}/enrich"
```

![alt text](docs/photo/persons-id-enrich-post.png)

5. Обновление

```bash
curl -X PUT "http://localhost/api/v1/persons/{UID}" \
  -H "Content-Type: application/json" \
  -d '{"name":"Dmitry","surname":"Ushakov","patronymic":"Alexeevich"}'
```

![alt text](docs/photo/persons-id-put.png)

5. Частичное

```bash
curl -X PATCH "http://localhost/api/v1/persons/{UID}" \
  -H "Content-Type: application/json" \
  -d '{"surname":"Ushakov"}'
```

![alt text](docs/photo/persons-id-patch.png)

6. Удаление

```bash
curl -X DELETE "http://localhost/api/v1/persons/{UID}"
```

![alt text](docs/photo/persons-id-delete.png)

## Используемые внешние сервисы

- agify.io – возраст по имени
- genderize.io – пол по имени
- nationalize.io – национальность по имени

## База данных

Таблица people:

| Поле                  | Тип                         | Описание                                                |
|-----------------------|-----------------------------|---------------------------------------------------------|
| `id`                  | UUID                        | Первичный ключ, уникальный идентификатор персоны        |
| `name`                | VARCHAR(100)                | Имя (обязательно)                                       |
| `surname`             | VARCHAR(100)                | Фамилия (обязательно)                                   |
| `patronymic`          | VARCHAR(100)                | Отчество (опционально)                                  |
| `age`                 | INTEGER                     | Возраст                                                 |
| `gender`              | VARCHAR(10)                 | Пол                                                     |
| `gender_probability`  | DECIMAL(5,4)                | Вероятность определения пола                            |
| `nationality`         | VARCHAR(2)                  | Код страны (национальность)                             |
| `nationality_probability` | DECIMAL(5,4)           | Вероятность определения национальности                  |
| `created_at`          | TIMESTAMP WITH TIME ZONE    | Дата и время создания записи                           |
| `updated_at`          | TIMESTAMP WITH TIME ZONE    | Дата и время последнего обновления записи              |

Миграции в `migrations` применяются автоматически при старте.