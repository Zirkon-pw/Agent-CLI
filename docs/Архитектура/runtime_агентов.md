# Runtime агентов

## Общая модель

В системе есть два явных способа запуска агента:

- `protocol_adapter` — внешний wrapper, который общается с supervisor через NDJSON protocol events/commands;
- `raw_cli` — logs-first fallback для обычного CLI без machine-readable stdout.

Pipeline устроен так:

```text
agents.yaml -> runtime spec -> driver registry -> supervisor -> session/artifact files
```

Это означает, что добавление нового агента делается через конфиг и driver-kind, а не через разрастание специальных веток в orchestration-коде.

## Как добавить нового агента

### Вариант 1: protocol wrapper

Используется, если агент умеет:

- принимать `stage_spec.json`;
- отдавать `hello`, `progress`, `artifact`, `stage_completed` и другие protocol events;
- участвовать в `review` и `handoff`, если это нужно.

В этом случае достаточно:

1. добавить wrapper executable;
2. прописать агента в `agents.yaml` как `runtime.kind: protocol_adapter`;
3. указать `runtime.exec`;
4. перечислить `runtime.supported_stages`;
5. при необходимости указать `runtime.child_cli`.

### Вариант 2: raw CLI fallback

Используется, если агент умеет только обычный stdout/stderr и exit code.

Такой агент:

- подходит для `execute` и `validate_fix`;
- не участвует в protocol-driven `clarification`, `review`, `handoff`;
- сохраняет `raw.stdout.log`, `raw.stderr.log` и `runtime_errors.log`.

Для подключения достаточно:

1. указать `runtime.kind: raw_cli`;
2. задать `runtime.exec.command` и `runtime.exec.args`;
3. ограничить `runtime.supported_stages` до реально поддерживаемых стадий.

## Что не должен делать implementer

- не добавлять vendor-specific ветки в supervisor для каждого нового агента;
- не возвращаться к flat-schema полям вроде `command`, `adapter_command`, `transport`;
- не пытаться парсить произвольный human stdout как protocol stream.

Если агенту нужен новый способ запуска, это должно оформляться как новый driver-kind, а не как ad-hoc исключение внутри существующих веток.
