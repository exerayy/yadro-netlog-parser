# YADRO Netlog Parser

Микросервис для парсинга логов InfiniBand сети, агрегации топологии и предоставления REST API.

## Запуск

```bash
docker compose up --build
```

Сервис поднимется на `http://localhost:8080`

Swagger документация на `http://localhost:8080/swagger/index.html`

Архивы кладутся в `data/` (папка монтируется в контейнер)

## Curl-запросы

```bash
curl -X POST http://localhost:8080/api/v1/parse \
-H "Content-Type: application/json" \
-d '{"path": "data/log.zip"}'
```

```bash
curl -X GET http://localhost:8080/api/v1/topology/1
```

```bash
curl -X GET http://localhost:8080/api/v1/node/1
```

```bash
curl -X GET http://localhost:8080/api/v1/port/1
```

```bash
curl -X GET http://localhost:8080/api/v1/log/1
```

## Архитектура

Проект построен по паттерну **Ports & Adapters** (Hexagonal Architecture):

### Core (Ядро)
- **parser** - парсинг архива с файлами `ibdiagnet2.db_csv` и `ibdiagnet2.sharp_an_info`
- **topology** - построение графа топологии, агрегация и выдача данных
- Не зависит от внешних фреймворков и БД
- Определяет интерфейсы (`ports.go`), которые реализуют адаптеры

### Adapters (Адаптеры)
- **http** - HTTP хендлеры, сериализация JSON
- **db** - реализация интерфейсов ядра для работы с PostgreSQL

## База данных

### Таблицы

| Таблица | Назначение |
|---------|------------|
| `logs` | Мета-информация о загруженных логах |
| `nodes` | Узлы сети (host/switch) |
| `ports` | Порты узлов с характеристиками |
| `nodes_info` | Расширенная информация об узлах |

### Миграции
Применяются автоматически при запуске приложения.

## API

### Формат ответа
Все ответы в JSON.

### Обработка ошибок

| HTTP код | Описание |
|----------|----------|
| 200 | Успешный запрос |
| 400 | Неверные параметры запроса |
| 404 | Ресурс не найден |
| 499 | Запрос отменен клиентом |
| 500 | Внутренняя ошибка сервера |

## Принцип построения топологии

### Источники данных
- **ibdiagnet2.db_csv** - узлы, порты, характеристики коммутаторов
- **ibdiagnet2.sharp_an_info** - параметры SHARP для коммутаторов

### Узлы
- `NodeType = 1` → **host** (сервер с HCA)
- `NodeType = 2` → **switch** (InfiniBand коммутатор)

### Порты
- **Management порт** (`PortNum = 0`) - содержит LID узла, не участвует в передаче данных
- **Data порты** (`PortNum > 0`, `PortState = 4`) - активные порты для построения связей

### LID (Local Identifier)
Назначается Subnet Manager каждому узлу. Используется для:
- Идентификации узла в сети
- Определения топологических связей

## Алгоритм построения связей

### 1. Сбор LID
LID собирается из двух источников:
- Management портов (`PortNum = 0`, `PortState = 0`)
- Data портов с ненулевым LID

Приоритет у management порта.

### 2. Связи host → switch
Хост соединяется со свитчем, имеющим минимальную разницу LID:
host1 (LID=1) → switch3 (LID=11) // разница 10

### 3. Связи switch → switch
Два критерия соединения:

**Прямое соединение** - общий LID у портов разных свитчей:
switch1:port5 (LID=10) → switch2:port3 (LID=10)

**Топологическая близость** - разница LID ≤ 10:
switch3 (LID=11) → switch4 (LID=12) // разница 1
switch4 (LID=12) → switch2 (LID=13) // разница 1
switch2 (LID=13) → switch1 (LID=22) // разница 9

Порог 10 выбран эмпирически для охвата многоуровневых топологий без ложных связей.

### 4. Дубликаты связей
Каждая пара узлов соединяется один раз. Уникальность определяется ключом:
"меньший_GUID:порт-больший_GUID:порт"

## Группы узлов

Автоматически создаются группы:
- **All Nodes** - все узлы
- **Hosts** - только хосты (`NodeType = 1`)
- **Switches** - только свитчи (`NodeType = 2`)
- **Product: <name>** - группировка по `ProductName` из `SYSTEM_GENERAL_INFORMATION`

## Ограничения
- Топология строится на основе LID, назначенных Subnet Manager
- Связи являются логическими, не физическими (кабель не отслеживается)
- Изолированные узлы (разница LID > 10 без прямых соединений) не включаются в граф
- Точность зависит от корректности назначения LID в сети

## Пример ответа API
Полученная топология из архива данного в задании
```json
{
  "log_id": 1,
  "nodes": [
    {
      "node_id": 21,
      "node_guid": "host1",
      "node_desc": "HOST_1",
      "node_type": "host",
      "num_ports": 1,
      "lid": 1
    },
    {
      "node_id": 22,
      "node_guid": "switch1",
      "node_desc": "SWITCH_1",
      "node_type": "switch",
      "num_ports": 65,
      "product_name": "Gorilla",
      "lid": 22
    },
    {
      "node_id": 23,
      "node_guid": "switch2",
      "node_desc": "SWITCH_2",
      "node_type": "switch",
      "num_ports": 65,
      "product_name": "Gorilla Prod",
      "lid": 13
    },
    {
      "node_id": 24,
      "node_guid": "switch3",
      "node_desc": "SWITCH_3",
      "node_type": "switch",
      "num_ports": 65,
      "product_name": "Gorilla CLust",
      "lid": 11
    },
    {
      "node_id": 25,
      "node_guid": "switch4",
      "node_desc": "SWITCH_4",
      "node_type": "switch",
      "num_ports": 65,
      "product_name": "Gorilla",
      "lid": 12
    }
  ],
  "edges": [
    {
      "from_node_guid": "host1",
      "from_port_num": 1,
      "to_node_guid": "switch3",
      "to_port_num": 1,
      "link_width": 2,
      "link_speed": 2048
    },
    {
      "from_node_guid": "switch2",
      "from_port_num": 1,
      "to_node_guid": "switch3",
      "to_port_num": 1,
      "link_width": 2,
      "link_speed": 2052
    },
    {
      "from_node_guid": "switch2",
      "from_port_num": 1,
      "to_node_guid": "switch4",
      "to_port_num": 1,
      "link_width": 2,
      "link_speed": 2052
    },
    {
      "from_node_guid": "switch3",
      "from_port_num": 1,
      "to_node_guid": "switch4",
      "to_port_num": 1,
      "link_width": 2,
      "link_speed": 2052
    }
  ],
  "groups": [
    {
      "group_name": "All Nodes",
      "node_guids": [
        "host1",
        "switch1",
        "switch2",
        "switch3",
        "switch4"
      ],
      "count": 5
    },
    {
      "group_name": "Switches",
      "node_guids": [
        "switch1",
        "switch2",
        "switch3",
        "switch4"
      ],
      "count": 4
    },
    {
      "group_name": "Hosts",
      "node_guids": [
        "host1"
      ],
      "count": 1
    },
    {
      "group_name": "Product: Gorilla",
      "node_guids": [
        "switch1",
        "switch4"
      ],
      "count": 2
    },
    {
      "group_name": "Product: Gorilla Prod",
      "node_guids": [
        "switch2"
      ],
      "count": 1
    },
    {
      "group_name": "Product: Gorilla CLust",
      "node_guids": [
        "switch3"
      ],
      "count": 1
    }
  ]
}
```