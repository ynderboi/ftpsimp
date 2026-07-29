# ftpsimp

Обмен файлами по Wi‑Fi в локальной сети: браузерный интерфейс для просмотра, загрузки, скачивания, создания папок и удаления файлов.

Репозиторий: [github.com/ynderboi/ftpsimp](https://github.com/ynderboi/ftpsimp)

**Подробная инструкция:** [docs/GUIDE.md](docs/GUIDE.md)

## Безопасность

По умолчанию включена **PIN-аутентификация**: при старте печатается 6-значный PIN (или берётся из `-pin` / настроек). Без PIN API недоступен.

| Защита | Поведение |
|--------|-----------|
| PIN / cookie-сессия | Все `/api/*` кроме `status` и `login` |
| Смена корневой папки | Только с **хоста** (loopback или LAN‑IP этой машины; на Android — в приложении) |
| Read-only | `-readonly` — только просмотр и скачивание |
| Без auth | `-open` — **только** в полностью доверенной сети |
| Upload | Не перезаписывает файлы без подтверждения (`overwrite`) |

Трафик идёт по обычному **HTTP** в LAN. Не выставляйте порт в интернет.

## Desktop (Go)

Требования: [Go](https://go.dev/) 1.22+.

```bash
go build -o ftpsimp.exe .
```

Запуск:

```bash
./ftpsimp.exe
./ftpsimp.exe -port 8080 -dir "C:\Users\You\Documents"
./ftpsimp.exe -pin 482910
./ftpsimp.exe -readonly
./ftpsimp.exe -open
```

| Флаг | Описание |
|------|----------|
| `-port` | HTTP-порт (`0` = из настроек, по умолчанию 8080) |
| `-dir` | Общая папка (пусто = из настроек / Documents) |
| `-pin` | Фиксированный PIN (сохраняется в config; иначе генерируется при старте) |
| `-readonly` | Только list/download (на этот запуск; в config — поле `readOnly`) |
| `-open` | Отключить PIN на этот запуск (или `"open": true` в config) |
| `-plain` | Классический вывод без интерактивного TUI |

После старта в терминале открывается **host panel** (TUI): ASCII‑лого, статус, PIN, сессии и горячие клавиши:

| Клавиша | Действие |
|---------|----------|
| `S` | Start / Stop HTTP‑сервера |
| `1` | Сменить корневую папку |
| `2` | Toggle read-only |
| `3` | Toggle auth (PIN) |
| `4` | Новый случайный PIN |
| `5` | Задать PIN вручную |
| `6` | Обновить LAN‑адреса |
| `Q` | Остановить сервер |

Настройки сохраняются в каталоге конфигурации пользователя (`…/ftpsimp/config.json`). Корневую папку также можно сменить в веб‑UI (только с хоста) или в TUI клавишей `1`.

После запуска откройте на другом устройстве `http://<LAN-IP>:8080` и введите PIN из host panel.

## Android

Откройте каталог [`android/`](android/) в Android Studio или соберите из командной строки:

```bash
cd android
.\gradlew.bat assembleDebug
```

APK: `android/app/build/outputs/apk/debug/`.

При запуске сервера на экране приложения показывается PIN для входа с другого устройства.

## Windows installer

Нужны Go и [Inno Setup 6](https://jrsoftware.org/isinfo.php).

```bash
cd installer
build.bat
```

Установщик появится в `dist/`.

## Лицензия

[MIT](LICENSE)
