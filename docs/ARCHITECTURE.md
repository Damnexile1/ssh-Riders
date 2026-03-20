# SSH Riders MVP Architecture

## 1. Краткий overview проекта
SSH Riders — real-time multiplayer terminal-игра, в которой игроки подключаются к gateway, попадают в комнату и управляют райдерами на общей grid-based арене. Каждый райдер двигается непрерывно, оставляет за собой след и выбывает при столкновении со стеной, препятствием или следом. Сервер комнаты authoritative: клиент отправляет только input, а весь расчёт мира, коллизий и победителя происходит на сервере комнаты. Gateway отвечает за SSH/session UX и lobby flow, orchestrator — за room registry и manifest-based lifecycle, room server — за игровой цикл и ASCII/ANSI frame generation. Такой формат отлично подходит под SSH: терминал уже является клиентом, а ASCII-рендер делает проект и зрелищным, и лёгким для запуска на любой машине. Для защиты проект выглядит сильно за счёт необычного UI-слоя, container-per-room архитектуры и понятного real-time loop.

## 2. Почему именно эта игра — удачный выбор
Tron-like light cycles дают высокий wow-effect при очень ограниченной механической сложности: всего 4 направления, grid movement, простые collision rules и мгновенно считываемый геймплей. Это намного реалистичнее для 2–4 дней, чем terminal fighting game с хитбоксами, анимациями, latency compensation и большим набором состояний. В ASCII/ANSI такая игра выглядит органично: арена, следы, райдеры, countdown, scoreboard и spectator mode хорошо читаются даже в моноширинном терминале. При этом архитектурно игра естественно ложится на authoritative room server и shared terminal world.

## 3. Архитектура решения
- **SSH Gateway / Lobby**: принимает SSH-сессии, создаёт isolated terminal session на игрока, показывает lobby, запрашивает комнату и проксирует input/render между игроком и room server.
- **Room Orchestrator / Room Manager**: хранит room registry, читает room manifests, создаёт room containers, отслеживает heartbeat/idle TTL и удаляет пустые комнаты.
- **Room Game Server**: authoritative game loop, input buffer, collision detection, round lifecycle, render snapshots/diffs, spectator mode.
- **Metadata / Room Registry / Storage**: для MVP — файл registry.json + in-memory registry в orchestrator; позже можно вынести в Redis/Postgres.

### Player flow
1. Игрок подключается к SSH Gateway.
2. Gateway открывает isolated session и показывает lobby.
3. Gateway запрашивает у orchestrator список комнат.
4. Игрок выбирает комнату или инициирует создание новой по manifest.
5. Gateway получает адрес room server и регистрирует игрока через join endpoint.
6. Игрок начинает отправлять только input events, а room server присылает render frames.

### Room lifecycle
1. Orchestrator получает create-room request с путём до manifest.
2. Manifest описывает image, сетевые параметры, tick rate, размеры арены и лимиты игроков.
3. Orchestrator поднимает container комнаты и регистрирует её в registry.
4. Room server периодически heartbeat'ит в orchestrator.
5. Если в комнате 0 игроков дольше idle TTL — orchestrator останавливает container.

## 4. ASCII-схема компонентов
```text
+------------------+      SSH       +-----------------------+
| Player Terminal  |  ------------> | SSH Gateway / Lobby   |
| ssh user@host    |                | - isolated session    |
+------------------+                | - room selection      |
        ^                           | - input/render bridge |
        | ASCII/ANSI frames         +-----------+-----------+
        |                                       |
        | REST/JSON                             | REST/JSON
        |                                       v
        |                           +-----------------------+
        |                           | Room Orchestrator     |
        |                           | - room registry       |
        |                           | - manifest loader     |
        |                           | - room lifecycle      |
        |                           +-----------+-----------+
        |                                       |
        |                            spawns / tracks containers
        |                                       v
        |                           +-----------------------+
        +-------------------------  | Room Container        |
                                    | Room Game Server      |
                                    | - authoritative loop  |
                                    | - collision rules     |
                                    | - snapshots / frames  |
                                    +-----------------------+
```

## 5. Игровая модель SSH Riders
- Arena MVP: 48x20 клеток.
- Игроков в комнате: 2–6.
- Tick rate: 8 TPS.
- Countdown между раундами: 3 секунды.
- Игрок не может остановиться и не может моментально развернуться на 180 градусов.
- Каждый tick райдер смещается на 1 клетку и оставляет занятость в occupied grid.
- Смерть: столкновение с границей, obstacle, собственным следом, чужим следом.
- Победа: последний живой игрок.
- После смерти игрок остаётся spectator'ом до следующего раунда.
- Новый раунд стартует автоматически после короткого finished/countdown окна.
- Scoreboard показывает имя, очки и статус ALIVE/DEAD.

## 6. Игровой цикл
1. Tick loop идёт по фиксированному ticker на сервере комнаты.
2. Клиент отправляет только input event с желаемым направлением.
3. Сервер буферизует последний валидный input per player.
4. На tick сервер применяет buffered input, если он не является reverse turn.
5. Затем считает next position для каждого живого райдера.
6. Выполняет collision detection against borders + occupied cells.
7. Помечает eliminated riders.
8. Обновляет occupied grid и scoreboard.
9. Если живых 0 или 1 — завершает раунд, начисляет score победителю и запускает новый round countdown.
10. Render pipeline строит ASCII full-frame; для MVP этого достаточно, позже можно добавить diff renderer.
11. Disconnect обрабатывается как leave: игрок удаляется из room state, а его session закрывается.

## 7. Структура репозитория
```text
cmd/gateway
cmd/room
cmd/orchestrator
internal/app/gateway
internal/app/room
internal/app/orchestrator
internal/config
internal/domain
internal/game
internal/render
internal/transport/internalapi
internal/transport/sshsession
pkg/logx
deployments/gateway
deployments/room
deployments/orchestrator
manifests
docs
README.md
Makefile
docker-compose.yml
```

## 8. Модель данных
Ключевые сущности: Player, Session, Room, RoomState, Arena, Position, Direction, OccupiedCell, RoundState, ScoreBoard, RoomManifest, InputEvent, RenderFrame. Их стартовые Go-структуры лежат в `internal/domain/models.go`.

## 9. Сетевое взаимодействие
Для MVP выбран **REST/JSON по внутренней сети контейнеров**. Почему не gRPC: быстрее развернуть, легче дебажить curl'ом, проще для hackathon-grade orchestrator/gateway integration. Почему не websocket: room authoritative и push-only transport можно добавить позже; на старте достаточно polling frame endpoint + POST input. События: `create room`, `list rooms`, `join room`, `input event`, `state snapshot`, `render frame`, `register room`.

## 10. Контейнеризация и запуск
- Отдельные Dockerfile для gateway, room и orchestrator.
- `docker-compose.yml` поднимает локальный stack.
- `manifests/room-alpha.yaml` задаёт стартовую комнату.
- В полноценной версии orchestrator будет запускать `docker run` / Docker API по manifest и гасить room container по idle TTL.

## 11. MVP roadmap на 2–4 дня
1. Gateway skeleton + hello screen.
2. Lobby + room registry.
3. Single room authoritative loop.
4. Multiplayer join/input/frame flow.
5. Room-per-container + manifests.
6. Polish: colors, spectator UX, kill feed, cleanup.

## 12. Кодовый каркас проекта
Skeleton включает три entrypoint'а (`gateway`, `room`, `orchestrator`), конфиги, domain models, room engine, ASCII renderer, join/input/state/frame endpoints, graceful shutdown и local terminal adapter как SSH session abstraction. Когда станет доступна dependency install, local adapter заменяется на `wish` без ломки game/domain слоя.

## 13. README
README содержит краткое описание игры, архитектуры, требования, docker compose запуск, локальный запуск, управление, структуру проекта и roadmap.

## 14. Быстрые фичи для защиты
1. ANSI colors per player — low cost, after MVP.
2. Spectator mode HUD — low cost, MVP+.
3. Kill feed — low cost, MVP+.
4. Shrinking arena — medium, after MVP.
5. Speed boost pickup — medium, after MVP.
6. Bots — medium, after MVP.
7. Replay log — medium, after MVP.
8. Hot room cleanup — low, should be in MVP polish.
9. Auto-close empty rooms — low, should be in MVP polish.
10. Leaderboard across rooms — medium, after MVP.

## 15. Почему решение хорошо выглядит перед жюри
Проект демонстрирует нестандартный, но полностью инженерный подход: SSH как transport/UI layer, authoritative real-time loop, container-per-room deployment model, manifest-based startup и понятная модульная архитектура. Это не "игрушечная" демка, а база для реальной multiplayer platform: gateway, orchestrator и room server разделены по ответственности, игровой state централизован и честно считается на сервере, а UI строится из переносимого ASCII/ANSI renderer. При этом scope остаётся хакатонно-реалистичным.
