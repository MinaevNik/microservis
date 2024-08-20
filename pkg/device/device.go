package device

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/exec"
    "strings"

    "github.com/jochenvg/go-udev"
)

// CreateMediaDirectory создаёт директорию /media, если она отсутствует
func CreateMediaDirectory() error {
    if _, err := os.Stat("/media"); os.IsNotExist(err) {
        err := os.MkdirAll("/media", 0755)
        if err != nil {
            return fmt.Errorf("failed to create /media directory: %w", err)
        }
    }
    return nil
}

// cleanDeviceName очищает имя устройства от лишних символов
func cleanDeviceName(name string) string {
    return strings.TrimLeft(name, "│─└├")
}

// mountDevice монтирует устройство
func mountDevice(device string) {
    device = cleanDeviceName(device)  // Очистка имени устройства
    mountPoint := fmt.Sprintf("/media/%s", device)
    err := os.MkdirAll(mountPoint, 0755)
    if err != nil {
        log.Fatalf("Failed to create mount point %s: %v", mountPoint, err)
    }

    devicePath := fmt.Sprintf("/dev/%s", device)
    fmt.Printf("Attempting to mount %s to %s\n", devicePath, mountPoint)
    cmd := exec.Command("sudo", "mount", devicePath, mountPoint) // Использование sudo для монтирования
    output, err := cmd.CombinedOutput()
    if err != nil {
        log.Fatalf("Failed to mount device %s: %v\n%s", devicePath, err, output)
    }

    fmt.Printf("Device %s mounted at %s\n", devicePath, mountPoint)
}

// Start запускает мониторинг устройств с использованием udev
func Start() {
    err := CreateMediaDirectory()
    if err != nil {
        log.Fatalf("Failed to create /media directory: %v", err)
    }

    udev := udev.Udev{}
    monitor := udev.NewMonitorFromNetlink("udev")
    monitor.FilterAddMatchSubsystemDevtype("block", "disk")

    ctx := context.Background() // Создаем контекст
    deviceChan, errChan, err := monitor.DeviceChan(ctx)
    if err != nil {
        log.Fatalf("Failed to start device monitor: %v", err)
    }

    log.Println("Device monitoring started...")

    for {
        select {
        case device := <-deviceChan:
            if device.Action() == "add" && device.IsInitialized() {
                log.Printf("Device added: %s", device.Syspath())
                mountDevice(device.Devnode())
            }
        case err := <-errChan:
            log.Printf("Device monitor error: %v", err)
        }
    }
}