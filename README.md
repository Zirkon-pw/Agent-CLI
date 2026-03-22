# agentctl

CLI-инструмент для управления инженерными задачами через AI-агентов. Заменяет чат-взаимодействие с LLM на структурированный pipeline: создание задачи, сборка контекста, выполнение агентом, валидация, ревью.

## Возможности

- **Формализованные задачи** — YAML-спецификации с целью, скоупом, ограничениями и критериями валидации
- **Шаблоны поведения** — 5 встроенных шаблонов (`clarify_if_needed`, `plan_before_execution`, `strict_executor`, `research_only`, `review_only`) + пользовательские
- **Мульти-агентность** — поддержка Claude, Codex, Qwen через их CLI
- **Уточнения** — структурированный YAML-flow вместо свободного чата
- **Валидация** — два режима: `simple` (pass/fail) и `full` (автоматическое исправление агентом, до N ретраев)
- **Наблюдаемость** — `ps`, `inspect`, `logs`, `events`, `watch` для мониторинга выполнения
- **Управление** — `stop`, `kill`, `pause`, `resume`, `cancel` для контроля запусков

## Требования

- Go 1.21+
- Один из CLI-агентов: `claude`, `codex`, `qwen`

## Сборка

```bash
# Клонировать репозиторий
git clone https://github.com/docup/agentctl.git
cd agentctl

# Собрать бинарник
make build
# Бинарник: build/agentctl

# Или установить в $GOPATH/bin
make install

# Полная проверка (tidy + fmt + vet + build)
make all
```

### Кросс-компиляция

```bash
make release
# Бинарники для linux/darwin/windows (amd64/arm64) в build/
```

## Быстрый старт

```bash
# 1. Инициализация проекта
agentctl init

# 2. Создание draft-задачи
agentctl task create

# 3. Конфигурация задачи
agentctl task update TASK-001 \
  --title "Рефакторинг auth модуля" \
  --goal "Вынести логику авторизации в отдельный сервисный слой" \
  --agent claude \
  --add-template clarify_if_needed

# 4. Запуск
agentctl task run TASK-001

# 5. Проверка результатов
agentctl task inspect TASK-001
agentctl result show TASK-001
agentctl result diff TASK-001

# 6. Принятие или отклонение
agentctl task accept TASK-001
agentctl task reject TASK-001 --reason "не покрыто тестами"
```

## Команды

### Задачи

| Команда | Описание |
|---------|----------|
| `task create` | Создать draft-задачу, в том числе пустую |
| `task update` | Донастроить задачу в статусе draft |
| `task run` | Запустить выполнение |
| `task resume` | Возобновить после паузы/стопа/уточнения |
| `task rerun` | Перезапустить задачу |
| `task list` | Список всех задач |
| `task inspect` | Детальная информация о задаче |
| `task ps` | Активные запуски |
| `task logs` | Логи выполнения (`-f` для follow) |
| `task events` | События жизненного цикла |
| `task watch` | Live-мониторинг |
| `task stop` | Мягкая остановка |
| `task kill` | Принудительная остановка |
| `task pause` | Пауза |
| `task cancel` | Отмена (для незапущенных) |
| `task accept` | Принять результат |
| `task reject` | Отклонить результат |
| `task route` | Переназначить агента |

### Шаблоны и уточнения

| Команда | Описание |
|---------|----------|
| `template list --builtin` | Встроенные шаблоны |
| `template show <id>` | Детали шаблона |
| `template add <path>` | Добавить пользовательский шаблон |
| `clarification generate` | Создать запрос на уточнение |
| `clarification show` | Показать ожидающий запрос |
| `clarification attach` | Прикрепить ответ |

### Прочее

| Команда | Описание |
|---------|----------|
| `guidelines add/list/show` | Управление гайдлайнами проекта |
| `result show/diff` | Просмотр результатов выполнения |
| `topics <topic>` | Справка по темам (`task`, `template`, `clarification`, `validation`, `workflow`) |

## Структура `.agentctl/`

```
.agentctl/
├── config.yaml          # Конфигурация проекта
├── agents.yaml          # Определения агентов
├── routing.yaml         # Правила маршрутизации
├── tasks/               # Спецификации задач (YAML)
├── templates/prompts/   # Пользовательские шаблоны
├── guidelines/          # Гайдлайны проекта (Markdown)
├── clarifications/      # Файлы уточнений
├── context/             # Собранные контекст-паки
├── runs/                # Артефакты выполнения
├── runtime/             # Состояние активных запусков
└── reviews/             # Решения по ревью
```

## Валидация

Два режима в конфиге задачи:

```yaml
validation:
  mode: full        # simple | full
  max_retries: 3    # только для full, по умолчанию 3
  commands:
    - go build ./...
    - go test ./tests/...
```

- **simple** — команды выполняются, exit 0 = pass, иначе fail
- **full** — при ошибке результат отправляется агенту на исправление, до `max_retries` попыток

## Создание и настройка задач

`task create` больше не требует обязательных `--title` и `--goal`. Команда может создать пустую draft-задачу, которую затем можно постепенно заполнить через `task update`.

Примеры:

```bash
# Пустая draft-задача
agentctl task create

# Частичное создание
agentctl task create --title "Подготовить auth refactor"

# Донастройка перед запуском
agentctl task update TASK-001 \
  --goal "Вынести логику авторизации в отдельный сервисный слой" \
  --agent claude \
  --add-template clarify_if_needed \
  --add-allowed-path internal/service/auth \
  --add-must-read README.md

# Расширенные правки через dot-path
agentctl task update TASK-001 \
  --set validation.mode=full \
  --add validation.commands="go test ./..." \
  --set runtime.max_execution_minutes=30
```

Перед `task run` и `task resume` у задачи обязательно должны быть заполнены `title` и `goal`. Если `agent` или built-in шаблоны не заданы, они будут подставлены из project config во время запуска и сохранены обратно в task YAML.

## Makefile

```bash
make build        # Собрать бинарник
make install      # Установить в $GOPATH/bin
make all          # tidy + fmt + vet + build
make test         # Все тесты из tests/
make test-cover   # Покрытие internal/* unit/integration/runtime тестами из tests/
make lint         # Линтер (golangci-lint)
make release      # Кросс-компиляция
make clean        # Очистка
make help         # Справка
```

## Архитектура

Проект построен по слоистой архитектуре:

```
cmd/agentctl/         → точка входа
internal/
  cli/                → команды (cobra)
  app/                → use cases (command/query + DTO)
  core/               → доменная модель (task, run, template, clarification)
  service/            → сервисы оркестрации (taskrunner, validation, prompting)
  infra/              → инфраструктура (fsstore, executor, runtime, events)
  config/             → конфигурация и встроенные шаблоны
  bootstrap/          → DI-wiring
```

Подробная документация — в директории `Docs/`.

## Лицензия

См. [LICENSE](LICENSE).
