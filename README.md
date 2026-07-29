# ftpsimp

Обмен файлами по Wi‑Fi в локальной сети: браузерный интерфейс для просмотра, загрузки, скачивания, создания папок и удаления файлов.

Репозиторий: [github.com/ynderboi/ftpsimp](https://github.com/ynderboi/ftpsimp)

## Важно по безопасности

Сервер **без аутентификации**, работает по обычному **HTTP**, CORS открыт. Любой в той же сети может читать и менять файлы в выбранной папке. Используйте только в **доверенной** локальной сети.

## Desktop (Go)

Требования: [Go](https://go.dev/) 1.22+.

```bash
go build -o ftpsimp.exe .
```

Запуск:

```bash
./ftpsimp.exe
./ftpsimp.exe -port 8080 -dir "C:\Users\You\Documents"
```

| Флаг   | Описание                                      |
|--------|-----------------------------------------------|
| `-port`| HTTP-порт (`0` = из настроек, по умолчанию 8080) |
| `-dir` | Общая папка (пусто = из настроек / Documents) |

Настройки сохраняются в каталоге конфигурации пользователя (`…/ftpsimp/config.json`). Корневую папку также можно сменить в веб-интерфейсе → Настройки.

После запуска откройте на другом устройстве адрес вида `http://<LAN-IP>:8080`.

## Android

Откройте каталог [`android/`](android/) в Android Studio или соберите из командной строки:

```bash
cd android
.\gradlew.bat assembleDebug
```

APK: `android/app/build/outputs/apk/debug/`.

## Windows installer

Нужны Go и [Inno Setup 6](https://jrsoftware.org/isinfo.php).

```bash
cd installer
build.bat
```

Установщик появится в `dist/`.

## Лицензия

[MIT](LICENSE)
