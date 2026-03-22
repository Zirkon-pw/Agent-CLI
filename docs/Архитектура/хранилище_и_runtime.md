# Хранилище и runtime

## Файловая модель как основа системы

Для Agent CLI файловое хранилище это не просто persistence, а часть самого протокола исполнения.

Все ключевые состояния должны быть представлены файлами внутри `.agentctl/`.

Это важно по трем причинам:

- процесс воспроизводим без истории чата;
- состояние легко диагностировать вручную;
- runtime можно восстановить после падения процесса.

## Базовая структура `.agentctl/`

```text
.agentctl/
  config.yaml
  agents.yaml
  routing.yaml
  tasks/
  templates/
  guidelines/
  clarifications/
  context/
  runs/
  runtime/
  reviews/
```

## Что хранится в `tasks/`

В `tasks/` лежит task spec:

- id и goal;
- статус;
- scope;
- `prompt_templates`;
- ссылки на `.yml`-уточнения;
- runtime settings;
- validation settings.

Task file должен оставаться основным снимком постановки задачи, но не превращаться в dump всех связанных файлов.

## Что хранится в `clarifications/`

В `clarifications/` лежат отдельные `.yml`-уточнения:

- `clarification_request_001.yml`
- `clarification_001.yml`

Они не должны встраиваться телом в `task.yml`. В task spec должны храниться только ссылки на них.

## Что хранится в `context/`

В `context/` лежит уже собранный execution package:

- resolved task snapshot;
- selected templates;
- clarification files;
- guidelines;
- must-read files;
- include files;
- summaries;
- execution metadata.

Этот каталог нужен одновременно:

- агенту как готовый контекст;
- CLI как воспроизводимый снимок исполнения;
- разработчику как материал для аудита.

## Что хранится в `runs/`

`runs/` хранит execution artifacts по каждому запуску.

Рекомендуемая структура:

```text
.agentctl/runs/TASK-001/RUN-001/
  prompt.md
  prompt_template_lock.yml
  attached_clarifications.json
  runtime.json
  stdout.log
  stderr.log
  events.ndjson
  diff.patch
  validation.json
  result_summary.md
```

`validation.json` включает историю повторных попыток (retry history) при использовании режима `full` validation.

Такой набор файлов дает понятный и проверяемый след каждого run.

## Runtime registry

Каталог `runtime/` нужен для живых процессов и для recovery.

Удобная структура:

```text
.agentctl/runtime/
  active_runs.json
  TASK-001/
    runtime.json
    heartbeat.json
    control.signal
    lock
    events.ndjson
```

Здесь удобно держать:

- список активных run;
- heartbeat state;
- locks;
- stop и kill signals;
- session mapping;
- короткие runtime snapshots.

## Event model

Для наблюдения полезно хранить события отдельно от логов.

Хороший формат это `ndjson`, потому что он:

- легко дописывается построчно;
- удобен для `tail -f`;
- читается человеком;
- легко парсится из Go.

Пример событий:

```json
{"ts":"2025-01-01T10:00:00Z","task":"TASK-001","run":"RUN-001","event":"queued"}
{"ts":"2025-01-01T10:00:05Z","task":"TASK-001","run":"RUN-001","event":"context_prepared"}
{"ts":"2025-01-01T10:00:12Z","task":"TASK-001","run":"RUN-001","event":"clarification_requested"}
{"ts":"2025-01-01T10:05:00Z","task":"TASK-001","run":"RUN-001","event":"clarification_attached"}
```

## Runtime state machine

Runtime должен понимать, что разные task statuses означают разные operational режимы:

- `draft` и `queued` не имеют активного процесса;
- `preparing_context` уже занята оркестрацией, но еще не имеет активного агента;
- `running` имеет активный run и heartbeat;
- `needs_clarification` не должен иметь продолжающегося execution worker;
- `ready_to_resume` ждет нового или продолженного run;
- `paused`, `stopped`, `killed` имеют разную operational семантику;
- `validating` означает уже не агентное исполнение, а validation pipeline;
- `review` означает завершенный run, ожидающий решения.

## Locks и защита от гонок

Чтобы два процесса не запустили одну задачу одновременно, runtime должен использовать lock на уровень задачи.

Минимальная модель:

- перед `task run` создается lock;
- lock содержит `task_id`, `run_id`, `pid`, `started_at`;
- при штатном завершении lock снимается;
- при recovery stale lock сверяется с heartbeat и process state.

## Recovery после перезапуска

Runtime слой должен уметь:

- перечитать `active_runs.json`;
- определить stale runs по heartbeat;
- понять, остался ли живой worker;
- перевести зависшие задачи в предсказуемый статус;
- не потерять attached clarifications и события.

Это особенно важно для long-running задач и для сценариев `task watch`, `task ps`, `task inspect`.

## Что критично для устойчивости

Чтобы CLI на Go работал предсказуемо, runtime должен уметь:

- восстанавливать state после рестарта;
- не запускать два активных run на одну задачу;
- различать graceful stop и forced kill;
- хранить отдельно run artifacts и live runtime state;
- понимать, какие `.yml`-уточнения уже были прикреплены;
- корректно завершать задачу, если clarification requested во время исполнения.

## Что важно для тестирования

Для тестов полезно абстрагировать файловый backend и runtime backend.

Все автоматические тесты, фикстуры и test support стоит хранить в отдельной корневой директории `tests/`.

Минимальный набор тестов:

- state transition tests;
- store read/write tests;
- template compatibility tests;
- clarification attach tests;
- task run orchestration tests;
- stop и kill behavior tests;
- recovery tests после незавершенного run.

Рекомендуемая раскладка:

- `tests/unit/`
- `tests/integration/`
- `tests/runtime/`
- `tests/e2e/`
- `tests/fixtures/`
- `tests/support/`
