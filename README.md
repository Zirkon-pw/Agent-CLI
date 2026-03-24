# agentctl

CLI-инструмент для управления инженерными задачами через AI-агентов. Заменяет чат-взаимодействие с LLM на структурированный pipeline: создание задачи, сборка контекста, stage-based выполнение агентом, валидация, reviewer-stage и review.

## Возможности

- **Формализованные задачи** — YAML-спецификации с целью, скоупом, ограничениями и критериями валидации
- **Шаблоны поведения** — 5 встроенных шаблонов (`clarify_if_needed`, `plan_before_execution`, `strict_executor`, `research_only`, `review_only`) + пользовательские
- **Adapter runtime** — запуск внешних CLI-агентов через adapter wrappers с NDJSON-протоколом по stdio
- **Stage/session модель** — `execute -> clarification -> validate_fix -> reviewer -> handoff` внутри одного session lifecycle
- **Уточнения** — структурированный YAML-flow, который материализуется supervisor-ом из protocol events
- **Валидация** — два режима: `simple` (pass/fail) и `full` (автоматическое исправление агентом, до N ретраев)
- **Наблюдаемость** — `ps`, `inspect`, `logs`, `events`, `watch` для мониторинга выполнения
- **Управление** — `stop`, `kill`, `cancel` и повторный `task run` для контроля live-session и recovery

## Требования

- Go 1.21+
- Внешний CLI-агент с adapter wrapper, который поддерживает machine-readable streaming mode

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

# 2. Создание задачи
agentctl task create \
  --title "Рефакторинг auth модуля" \
  --goal "Вынести логику авторизации в отдельный сервисный слой" \
  --agent claude

# 3. Донастройка задачи
agentctl task update TASK-001 \
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
| `task create` | Создать draft-задачу (обязательны `--title` и `--goal`) |
| `task update` | Донастроить задачу в статусе draft |
| `task run` | Запустить или продолжить session pipeline |
| `task rerun` | Перезапустить задачу |
| `task list` | Список всех задач |
| `task inspect` | Детальная информация о задаче |
| `task ps` | Активные запуски |
| `task logs` | Session/stage логи (`--stage`, `--protocol`, `-f`) |
| `task events` | События жизненного цикла |
| `task watch` | Live-мониторинг |
| `task stop` | Мягкая остановка |
| `task kill` | Принудительная остановка |
| `task cancel` | Отмена (для незапущенных) |
| `task accept` | Принять результат |
| `task reject` | Отклонить результат |
| `task route` | Поставить handoff на другого агента |

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
| `result show/diff/list` | Просмотр результатов и артефактов выполнения |
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
├── runs/                # Session directories, stage history, protocol.ndjson, artifacts.json
├── runtime/             # Состояние активных session и control commands
└── reviews/             # Решения по ревью
```

Типичный session layout:

```text
.agentctl/runs/TASK-001/RUN-001/
├── metadata.json
├── session.json
├── protocol.ndjson
├── artifacts.json
└── stages/
    ├── STAGE-001/
    │   ├── stage_spec.json
    │   ├── prompt.md
    │   ├── adapter.stderr.log
    │   ├── raw.stdout.log
    │   ├── raw.stderr.log
    │   └── runtime_errors.log
    └── STAGE-002/
        └── review_prompt.md
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

## Взаимодействие с агентом

`agentctl` больше не запускает агент как одноразовую команду с одним `prompt` и финальным `stdout`.

Теперь runtime работает так:

1. Для задачи создается `RunSession`.
2. Supervisor планирует следующий `stage`.
3. Для stage пишется `stage_spec.json`.
4. Adapter wrapper запускает внешний CLI-агент и общается с ним через NDJSON:
   - `stdin` — control commands (`cancel`, `kill`, `ping` и другие protocol-level команды)
   - `stdout` — protocol events (`hello`, `progress`, `artifact`, `clarification_requested`, `review_report`, `stage_completed` и т.д.)
5. Supervisor пишет сырой поток в `protocol.ndjson`, поддерживает `artifacts.json`, обновляет `session.json` и materialize-ит YAML-файлы уточнений для пользователя.

## `agents.yaml`

Новая схема `agents.yaml` разделяет descriptive metadata и runtime-способ запуска агента.

Пример:

```yaml
agents:
  - id: claude
    role: executor
    metadata:
      specialization: [architecture_refactor, deep_analysis]
      strengths: [large_context_reasoning, architecture_review]
      speed: medium
      cost: high
      context_limit: large
      modes: [strict, research]
      tools: [filesystem, git]
    runtime:
      kind: protocol_adapter
      exec:
        command: claude-wrapper
        args: []
      supported_stages: [execute, validate_fix, review, handoff]
      control:
        cancel: true
        kill: true
        heartbeat: true
      protocol:
        version: v1
      child_cli:
        command: claude
        args: [-p]

  - id: qwen
    role: executor
    metadata:
      specialization: [bulk_tests, code_generation]
      strengths: [fast_generation, code_edits]
      speed: high
      cost: low
      context_limit: medium
      modes: [strict, fast]
      tools: [filesystem, terminal]
    runtime:
      kind: raw_cli
      exec:
        command: qwen
        args: []
      supported_stages: [execute, validate_fix]
      control:
        cancel: true
        kill: true
```

- `protocol_adapter` — полноценный stage-aware wrapper с NDJSON-протоколом.
- `raw_cli` — logs-first fallback для обычных CLI без machine-readable stdout.
- Старые поля `command`, `args`, `adapter_command`, `transport` и похожие больше не поддерживаются.

## Создание и настройка задач

`task create` требует обязательные `--title` и `--goal`. Дополнительные поля можно задать сразу при создании или позже через `task update`.

Примеры:

```bash
# Создание задачи с минимальными полями
agentctl task create \
  --title "Подготовить auth refactor" \
  --goal "Вынести логику авторизации в отдельный сервисный слой"

# Создание задачи с полной конфигурацией
agentctl task create \
  --title "Подготовить auth refactor" \
  --goal "Вынести логику авторизации в отдельный сервисный слой" \
  --agent claude \
  --template clarify_if_needed \
  --allowed-path internal/service/auth \
  --must-read README.md

# Донастройка через task update
agentctl task update TASK-001 \
  --add-template clarify_if_needed \
  --add-allowed-path internal/service/auth

# Расширенные правки через dot-path
agentctl task update TASK-001 \
  --set validation.mode=full \
  --add validation.commands="go test ./..." \
  --set runtime.max_execution_minutes=30
```

Если `agent` или built-in шаблоны не заданы, они будут подставлены из project config во время запуска и сохранены обратно в task YAML. При запуске также проверяется, что указанный агент существует в `agents.yaml`.

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
  infra/              → инфраструктура (fsstore, runtime, events)
  config/             → конфигурация и встроенные шаблоны
  bootstrap/          → DI-wiring
```

Подробная документация — в директории `docs/`.

## Лицензия

См. [LICENSE](LICENSE).
