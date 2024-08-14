
# Проект "MicroServis"

## Описание проекта

Проект "MicroServis" представляет собой серверное приложение, написанное на Go, которое выполняет несколько задач по настройке системы при запуске. Эти задачи включают настройку модуля реального времени (RTC), настройку сетевых параметров (Ethernet и WiFi), управление обновлением прошивки, автоматическое монтирование USB-устройств и создание systemd-сервиса, а также запуск HTTP сервера для обработки различных API запросов.

## Структура проекта

Проект состоит из следующих пакетов:

1. **main**
   - Точка входа в приложение. Настраивает RTC, Ethernet и WiFi, а затем запускает сервер API.

2. **api**
   - Отвечает за обработку HTTP запросов. Включает эндпоинты для работы с сетями, управления системой и обновления прошивки.

3. **shutdown**
   - Содержит функции для выключения и перезагрузки системы.

4. **wifi**
   - Управляет конфигурацией и операциями WiFi.

5. **ethernet**
   - Управляет конфигурацией и информацией Ethernet.

6. **rtc**
   - Настраивает модуль реального времени (RTC) на Raspberry Pi.

7. **update**
   - Обеспечивает функционал для обновления прошивки и отката на предыдущие версии.

8. **device**
   - Отвечает за автомонтирование USB-устройств и создание необходимых директорий для этого процесса.

9. **systemd**
   - Создает и управляет systemd-сервисом для автоматического монтирования USB-устройств и запуска соответствующего скрипта.

## Подробности пакетов

### main

Файл `main.go`:
- Настраивает RTC, Ethernet и WiFi при запуске программы.
- Запускает сервер API.

### api

Файл `api.go`:
- Обрабатывает HTTP запросы на получение списка сетей, подключение к сети, управление системой и обновление прошивки.
- Эндпоинты:
  - `GET /networks/all`: Получить список доступных сетей WiFi.
  - `POST /networks/connect`: Подключиться к выбранной сети WiFi.
  - `POST /shutdown`: Выключить систему.
  - `POST /reboot`: Перезагрузить систему.
  - `GET /usb/files`: Получить список ZIP-файлов на подключенных USB-устройствах с информацией о версиях файлов.
  - `POST /firmware/update`: Начать обновление прошивки, указав выбранный ZIP-файл.
  - `POST /firmware/rollback`: Откатить прошивку на предыдущую версию.

### shutdown

Файл `shutdown.go`:
- Функция `Shutdown() error`: Выключает систему.
- Функция `Reboot() error`: Перезагружает систему.

### wifi

Файл `wifi.go`:
- Функция `RunCommand(name string, args ...string) (string, error)`: Выполняет команду в shell.
- Функция `StopWpaSupplicant()`: Останавливает процесс `wpa_supplicant`.
- Функция `UpdateNetworkConfig(filePath, ssid, psk string) error`: Обновляет конфигурационный файл WiFi.
- Функция `ScanNetworks(wifiInterface string) ([]map[string]interface{}, error)`: Сканирует доступные сети WiFi.

### ethernet

Файл `ethernet.go`:
- Функция `RunCommand(name string, args ...string) (string, error)`: Выполняет команду в shell.
- Функция `GetEthernetInfo(interfaceName string) (string, string, string, string, error)`: Получает информацию о конфигурации Ethernet.
- Функция `UpdateEthernetConfig(filePath, ipAddr, netmask, gateway, dns string) error`: Обновляет конфигурационный файл Ethernet.
- Функция `ConfigureEthernet() error`: Выполняет настройку Ethernet и WiFi.

### rtc

Файл `rtc.go`:
- Функция `RunCommand(name string, args ...string) (string, error)`: Выполняет команду в shell.
- Функция `EnableI2C() error`: Включает интерфейс I2C.
- Функция `RemoveFakeHwClock() error`: Удаляет пакет `fake-hwclock`.
- Функция `UpdateHwClockSetScript() error`: Обновляет скрипт `hwclock-set`.
- Функция `SyncTime() error`: Синхронизирует время с RTC.
- Функция `ConfigureRTC() error`: Настраивает модуль RTC без перезагрузки.

### update

Файл `update.go`:
- Функция `GetUSBMountPoints() ([]string, error)`: Возвращает список смонтированных USB-устройств.
- Функция `FindValidFirmware(zipReader *zip.Reader) (*FirmwareInfo, error)`: Извлекает информацию о прошивке из ZIP-файла.
- Функция `UpdateFirmware(zipFilePath string, versionFilePath string, backupDir string) error`: Выполняет обновление прошивки.
- Функция `RollbackFirmware(backupDir string, installedVersions *InstalledVersionInfo) error`: Выполняет откат прошивки на предыдущую версию.

### device

Файл `device.go`:
- Функция `CreateMediaDirectory() error`: Создает директорию `/media`, если она отсутствует.
- Функция `CheckAndMountDevices() error`: Проверяет устройства и монтирует их, если они съемные и не смонтированы.
- Функция `StartDeviceMonitor()`: Запускает мониторинг устройств и их автомонтирование.

### systemd

Файл `systemd.go`:
- Функция `BuildSelf() error`: Компилирует сам себя.
- Функция `CreateSystemdService(serviceName, execPath string) error`: Создает и регистрирует systemd-сервис.
- Функция `Manage()`: Выполняет все необходимые шаги: компиляция, создание сервиса и запуск мониторинга устройств.

## Пример использования

1. Запустите программу:
   ```bash
   sudo ./servis
   ```

2. Используйте API для взаимодействия с системой:
   - Получить список сетей WiFi:
     ```bash
     curl -X GET http://localhost:4444/networks/all
     ```
   - Подключиться к сети WiFi:
     ```bash
     curl -X POST -d '{"name": "YourSSID", "password": "YourPassword"}' http://localhost:4444/networks/connect
     ```
   - Выключить систему:
     ```bash
     curl -X POST -d '{"comment": "shutdown now"}' http://localhost:4444/shutdown
     ```
   - Перезагрузить систему:
     ```bash
     curl -X POST -d '{"comment": "reboot now"}' http://localhost:4444/reboot
     ```
   - Получить список ZIP-файлов с версиями прошивки:
     ```bash
     curl -X GET http://localhost:4444/usb/files
     ```
   - Начать обновление прошивки:
     ```bash
     curl -X POST -d '{"selected_file": "/media/sda1/firmware.zip"}' http://localhost:4444/firmware/update
     ```
   - Откатить прошивку на предыдущую версию:
     ```bash
     curl -X POST http://localhost:4444/firmware/rollback
     ```
