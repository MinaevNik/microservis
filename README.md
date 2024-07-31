
# Проект "Servis"

## Описание проекта

Проект "Servis" представляет собой серверное приложение, написанное на Go, которое выполняет несколько задач по настройке системы при запуске. Эти задачи включают настройку модуля реального времени (RTC), настройку сетевых параметров (Ethernet и WiFi), а также запуск HTTP сервера для обработки различных API запросов.

## Структура проекта

Проект состоит из следующих пакетов:

1. **main**
   - Точка входа в приложение. Настраивает RTC, Ethernet и WiFi, а затем запускает сервер API.

2. **api**
   - Отвечает за обработку HTTP запросов. Включает эндпоинты для работы с сетями и управления системой.

3. **shutdown**
   - Содержит функции для выключения и перезагрузки системы.

4. **wifi**
   - Управляет конфигурацией и операциями WiFi.

5. **ethernet**
   - Управляет конфигурацией и информацией Ethernet.

6. **rtc**
   - Настраивает модуль реального времени (RTC) на Raspberry Pi.

## Установка и настройка

### Требования

- Raspberry Pi с установленной ОС Raspberry Pi OS
- Go 1.15 или выше
- Права суперпользователя для выполнения команд

### Сборка

1. Склонируйте репозиторий:
   ```bash
   git clone https://github.com/yourusername/servis.git
   cd servis
   ```

2. Скомпилируйте проект:
   ```bash
   go build -o servis
   ```

### Запуск

1. Убедитесь, что у вас есть необходимые права для выполнения системных команд.

2. Запустите приложение:
   ```bash
   sudo ./servis
   ```

## Подробности пакетов

### main

Файл `main.go`:
- Настраивает RTC, Ethernet и WiFi при запуске программы.
- Запускает сервер API.

### api

Файл `api.go`:
- Обрабатывает HTTP запросы на получение списка сетей, подключение к сети, выключение и перезагрузку системы.
- Эндпоинты:
  - `GET /networks/all`: Получить список доступных сетей WiFi.
  - `POST /networks/connect`: Подключиться к выбранной сети WiFi.
  - `POST /shutdown`: Выключить систему.
  - `POST /reboot`: Перезагрузить систему.

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


