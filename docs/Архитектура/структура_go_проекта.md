# Структура Go проекта

## Целевая раскладка каталогов

Ниже приведена рекомендуемая структура исходного кода Agent CLI на Go:

```text
cmd/
  agentctl/
    main.go

tests/
  unit/
  integration/
  runtime/
  e2e/
  fixtures/
  support/

internal/
  bootstrap/
    app.go
    wiring.go

  cli/
    root/
    help/
    task/
    template/
    clarification/
    result/
    guidelines/

  app/
    command/
    query/
    dto/

  core/
    task/
      entity.go
      status.go
      transitions.go
    run/
      entity.go
      status.go
    template/
      entity.go
      behavior.go
    clarification/
      request.go
      entity.go
    runtime/
      state.go
      signal.go
    review/
      decision.go
    validation/
      report.go

  service/
    workspace/
    contextpack/
    prompting/
    taskrunner/
    clarificationflow/
    runtimecontrol/
    validationrunner/

  infra/
    fsstore/
    executor/
    llm/
    runtime/
    events/
    logging/
    clock/

  config/
    loader/
    schema/
    builtin_templates/
```

## Назначение уровней

### `cmd/agentctl`

Точка входа в бинарник.

Здесь должно быть только:

- создание `context.Context`;
- вызов bootstrap;
- запуск root command;
- преобразование финальной ошибки в exit code.

### `internal/bootstrap`

Слой сборки приложения.

Он создает:

- config loader;
- stores;
- registries;
- services;
- root CLI command.

Это удобное место для dependency wiring, чтобы не раздувать `main.go`.

### `internal/cli`

CLI adapter layer.

Он содержит:

- дерево команд;
- флаги и их маппинг;
- help messages;
- printers для таблиц и summary;
- преобразование CLI input в application requests.

В этом слое не должна жить бизнес-логика задач.

### `internal/app`

Use case layer.

Здесь удобно держать application commands и queries:

- `CreateTask`
- `UpdateTaskTemplates`
- `RunTask`
- `ResumeTask`
- `GenerateClarification`
- `AttachClarification`
- `StopTask`
- `KillTask`
- `InspectTask`
- `ListActiveTasks`

Этот слой координирует domain и infrastructure.

### `internal/core`

Domain model.

Здесь лежат:

- типы;
- status machine;
- правила переходов;
- инварианты task, run, clarification и review.

Примеры инвариантов:

- нельзя продолжать задачу через `task run`, если она не в состоянии, пригодном для нового запуска;
- нельзя `attach clarification`, если нет pending request;
- нельзя `kill`, если нет активного run;
- нельзя запускать несовместимые built-in templates в одной задаче.

### `internal/service`

Сервисы, объединяющие несколько доменных и инфраструктурных операций:

- построение prompt contract;
- сбор context pack;
- orchestration clarification flow;
- запуск validation;
- управление runtime;
- сбор итоговых артефактов.

### `internal/infra`

Реализация внешних зависимостей:

- файловое хранилище `.agentctl/`;
- LLM executor adapters;
- process execution;
- runtime registry;
- event stream;
- structured logging.

### `internal/config`

Загрузка и валидация конфигурации:

- `config.yaml`
- `agents.yaml`
- `routing.yaml`
- built-in template catalog

### `tests`

Все автотесты проекта должны жить в отдельной корневой директории `tests/`.

Это осознанное проектное правило, даже если оно отличается от типичной go-практики с `_test.go` рядом с production-кодом.

Рекомендуемое деление:

- `tests/unit/` для быстрых сценариев на уровень пакетов и use cases;
- `tests/integration/` для проверок stores, runtime и executor adapters;
- `tests/runtime/` для recovery, heartbeat, locks, `ps`, `stop`, `kill`;
- `tests/e2e/` для полных CLI-сценариев;
- `tests/fixtures/` для sample tasks, templates, runs и workspace fixtures;
- `tests/support/` для test doubles, builders и вспомогательных утилит.

Такое правило упрощает навигацию по проекту и делает тестовый слой отдельной подсистемой.

## Как разложить CLI по пакетам

Внутри `internal/cli` полезно придерживаться схемы:

```text
internal/cli/
  root/
    command.go
  task/
    create.go
    update.go
    run.go
    control.go
    inspect.go
    ps.go
    stop.go
    kill.go
  template/
    list.go
    show.go
    add.go
  clarification/
    generate.go
    attach.go
    show.go
  help/
    topics.go
  result/
    show.go
    diff.go
```

Это дает понятное соответствие между CLI-деревом и файлами в коде.

## Как разложить stores и adapters

Для инфраструктуры подходит следующая схема:

```text
internal/infra/fsstore/
  tasks.go
  runs.go
  clarifications.go
  templates.go
  workspace.go

internal/infra/runtime/
  registry.go
  heartbeat.go
  signals.go
  recovery.go

internal/infra/llm/
  codex_executor.go
  claude_executor.go
  qwen_executor.go
```

Такая раскладка сохраняет простой и прямой маппинг между предметной сущностью и файловой реализацией.

## Как организовать тестовый каталог

Для тестовой директории полезно придерживаться схемы:

```text
tests/
  unit/
    app/
    core/
    service/
  integration/
    fsstore/
    runtime/
    llm/
  runtime/
    recovery/
    control/
  e2e/
    cli/
  fixtures/
    workspaces/
    tasks/
    templates/
  support/
    builders/
    fakes/
```

Это дает единое место для всех сценариев и не смешивает тестовые артефакты с production-пакетами.

## Почему такая структура подходит для Go

Эта схема соответствует практическому стилю Go:

- `cmd` отделяет entrypoint от остального кода;
- `tests` делает тестовый слой отдельным и предсказуемым;
- `internal` ограничивает внешнюю видимость;
- composition root не смешивается с CLI handlers;
- domain и use cases не смешиваются с файловой системой;
- инфраструктуру можно менять без переписывания task logic.

При таком подходе тесты в основном работают как black-box проверки поверх публичных конструкторов и интерфейсов пакетов.

## Что не стоит делать

Нежелательная структура для этого CLI:

- класть всю логику в `cmd/agentctl/main.go`;
- смешивать YAML parsing и orchestration внутри CLI handlers;
- читать и писать `.agentctl/` напрямую из каждой команды;
- хранить статусы как набор разбросанных `const string`;
- строить prompt из случайной конкатенации строк без отдельного builder-а;
- прятать built-in template logic внутри executor adapters.
- держать часть тестов в `tests/`, а часть рядом с production-кодом.
