package systemd

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"

	"servis/pkg/device"
)

// BuildSelf компилирует сам себя
func BuildSelf() error {
	cmd := exec.Command("go", "build", "-o", "automount", "main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to build the executable: %w", err)
	}
	fmt.Println("Build successful!")
	return nil
}

// CreateSystemdService создает и регистрирует systemd сервис
func CreateSystemdService(serviceName, execPath string) error {
	serviceFilePath := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)

	const serviceFileContent = `[Unit]
Description=Auto Mount USB Drives

[Service]
ExecStart=%s
Restart=always

[Install]
WantedBy=multi-user.target
`
	// Открытие файла для записи
	file, err := os.OpenFile(serviceFilePath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open service file: %w", err)
	}
	defer file.Close()

	// Запись содержимого файла
	_, err = file.WriteString(fmt.Sprintf(serviceFileContent, execPath))
	if err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Перезагрузка конфигурации systemd
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return fmt.Errorf("failed to reload systemd daemon: %w", err)
	}

	// Включение и запуск сервиса
	if err := exec.Command("systemctl", "enable", serviceName+".service").Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	if err := exec.Command("systemctl", "start", serviceName+".service").Run(); err != nil {
		return fmt.Errorf("failed to start service: %w", err)
	}

	return nil
}

// Manage выполняет все необходимые шаги: компиляция, создание сервиса и запуск мониторинга
func Manage() {
	// Автоматическая компиляция самого себя
	if err := BuildSelf(); err != nil {
		log.Fatalf("Build failed: %v", err)
	}

	execPath, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable path: %v", err)
	}
	execPath, err = filepath.Abs(execPath)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}

	serviceName := "automount"

	// Создание и запуск Systemd сервиса
	err = CreateSystemdService(serviceName, execPath)
	if err != nil {
		log.Fatalf("Failed to create systemd service: %v", err)
	}

	// Запуск автомонитора устройств
	device.StartDeviceMonitor()
}
