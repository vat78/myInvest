# myInvest

Локальное desktop-приложение для учёта инвестиций: Go, Wails, SQLite и
Svelte/TypeScript. Пока реализована техническая оболочка по спецификации
[000](docs/features/000-app-bootstrap.md): окно, связь UI с Go и безопасная
подготовка хранилища. Бизнес-таблиц и финансовых расчётов ещё нет.

## Среда

Закреплены Go 1.27.2 и Wails 2.15.0; версии фронтенда указаны в
`frontend/package.json`, транзитивные зависимости — в lock-файлах.
Использовались Node 26.3.1 и npm 11.16.0. Vite/plugin требуют Node
20.19+, 22.12+ либо 24+; для воспроизведения используйте проверенные версии.

На macOS нужны Xcode Command Line Tools (`xcode-select --install`).
На Windows нужны компилятор/зависимости Wails и WebView2; на Linux — GCC,
GTK3 и WebKitGTK. Точные платформенные требования:
[документация Wails](https://wails.io/docs/gettingstarted/installation/).
Проверенная ОС — **macOS 27.0.1, Apple Silicon (ARM64)**.
Windows, Linux и более ранние версии macOS в этой задаче не проверялись.
Cross compilation desktop-приложения между ОС не настроена.

## Установка и разработка

Из корня чистого checkout:

```sh
go install github.com/wailsapp/wails/v2/cmd/wails@v2.15.0
# Добавьте каталог `go env GOPATH`/bin в PATH.
go mod download
npm --prefix frontend ci
wails dev
```

`wails dev` запускает Vite и desktop-окно с Go backend. Простое `npm run dev`
запускает только фронтенд: вызовы Go работают внутри Wails.
Bindings в `frontend/wailsjs` генерирует Wails; они хранятся в Git, чтобы
проверка фронтенда была доступна после `npm ci` без генерации.

## Сборка

```sh
wails build
```

На macOS результат — `build/bin/myInvest.app`, открывается через
`open build/bin/myInvest.app`. На Windows/Linux результат находится в
`build/bin/`. Бинарник содержит ресурсы UI и SQL-миграции и не требует checkout.
Сборка локальная, без подписи, notarization и публикации.

## Данные и обновления

БД создаётся автоматически; cwd не влияет на путь:

| ОС | Путь |
|---|---|
| macOS | `~/Library/Application Support/myInvest/myinvest.sqlite3` |
| Windows | `%LOCALAPPDATA%\myInvest\myinvest.sqlite3` |
| Linux | `$XDG_DATA_HOME/myInvest/myinvest.sqlite3`, по умолчанию `~/.local/share/myInvest/myinvest.sqlite3` |

Для изолированного запуска можно указать **абсолютный** каталог:

```sh
MYINVEST_DATA_DIR=/tmp/myinvest-smoke wails dev
```

У release-бинарника действует тот же параметр окружения. Не подменяйте им
каталог рабочей БД для тестов. WAL и sidecar-файлы управляются SQLite.
Соединения закрываются при выходе; база при повторном запуске сохраняется.

Переносимый режим включается через `MYINVEST_PORTABLE=1`. На macOS файл
`myinvest.sqlite3` создаётся рядом с `.app`, на Windows/Linux и для отдельного
macOS-бинарника — рядом с исполняемым файлом. Например, на macOS:

```sh
MYINVEST_PORTABLE=1 /путь/myInvest.app/Contents/MacOS/myInvest
```

На Windows PowerShell задайте `$env:MYINVEST_PORTABLE = '1'`, затем запустите
`./myInvest.exe`; на Linux — `MYINVEST_PORTABLE=1 ./myInvest`.
`MYINVEST_PORTABLE=0` или отсутствие переменной сохраняет системный каталог.
`MYINVEST_DATA_DIR` имеет приоритет над переносимым режимом. Символические ссылки
разрешаются: используется каталог фактического бинарника. При `wails dev` это
каталог сгенерированного приложения; для разработки удобнее `MYINVEST_DATA_DIR`.
Каталог должен разрешать запись БД, WAL и резервных копий; при ошибке приложение
покажет сообщение и завершится без перехода в другой каталог. Существующая БД
автоматически не переносится. Для ручного переноса закройте приложение и сохраните
исходную БД вместе с имеющимися sidecar-файлами. При переносе приложения переносите
также БД, её sidecar-файлы и `backups/`.

Новые миграции добавляются как `migrations/0002_name.sql`, затем 0003 и т.д.
Применённые файлы не редактируются. Миграции не содержат управления транзакциями:
адаптер атомарно выполняет файл и записывает версию. Начальная миграция создаёт
только `schema_migrations`. SHA-256 позволяет обнаружить изменение истории;
запуск старой версии приложения с более новой схемой запрещён.

Перед обновлением существующей БД создаётся согласованная копия через
`VACUUM INTO` в `backups/myinvest-<UTC>-<случайный суффикс>.sqlite3` рядом с БД.
Копия включает данные из WAL. Ошибка копирования останавливает обновление.
Ошибка миграции откатывает только эту миграцию вместе с записью её версии;
ранее завершённые миграции сохраняются. Исходная БД не удаляется и не сбрасывается.
При ошибке инициализации приложение показывает сообщение и закрывается.
Копии не удаляются автоматически; восстановление выполняется отдельно,
при закрытом приложении и после сохранения исходных файлов.

## Проверки

```sh
npm --prefix frontend ci
npm --prefix frontend run check
npm --prefix frontend run build
# dist нужен go:embed; в чистом checkout сначала соберите фронтенд.
gofmt -w app.go app_test.go main.go internal/storage/sqlite/*.go migrations/embed.go
go test ./...
go vet ./...
wails build
```

Интеграционные тесты используют только временные SQLite-БД: повторный запуск,
порядок миграций, откат SQL и записи версии, внешние ключи на разных соединениях,
резервная копия с WAL, блокировка миграций при ошибке копирования, несовместимая
история и сохранение повреждённого файла. UI проверяется через реальное окно Wails;
отдельного набора frontend unit-тестов в этой минимальной оболочке нет.

Детали решений: [ADR 0001](docs/decisions/0001-app-bootstrap.md).

Проверка реализации 000 (2026-10-12): `gofmt`, `go test ./...`, `go vet ./...`,
`npm ci`, `npm run check`, `npm run build` и `wails build -nosyncgomod`.
CLI использовался из временного GOBIN, Go build cache также размещался в /tmp
из-за ограничений среды. `-nosyncgomod` сохраняет закреплённый go.mod;
версия CLI совпадает с зависимостью Wails. Проверены release-окно и успешный
Go-вызов в release и `wails dev`, запуск скопированного .app вне checkout,
повторный запуск и диалог
повреждённой тестовой БД. Линкер выдаёт предупреждение о macOS 13 для Go-объекта
при deployment target Wails 11; совместимость со старыми macOS не заявляется.
