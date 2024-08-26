package device

import (
    //"context"
    "fmt"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "strings"
    "time"

    "github.com/fsnotify/fsnotify"
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
    return strings.TrimLeft(filepath.Base(name), "│─└├")
}

// mountDevice монтирует устройство
func mountDevice(device string) {
    deviceName := cleanDeviceName(device)
    mountPoint := fmt.Sprintf("/media/%s", deviceName)

    // Создаем точку монтирования, если она не существует
    err := os.MkdirAll(mountPoint, 0755)
    if err != nil {
        log.Fatalf("Failed to create mount point %s: %v", mountPoint, err)
    }

    // Определяем, есть ли разделы у устройства
    cmd := exec.Command("lsblk", "-no", "MOUNTPOINT", device)
    output, err := cmd.Output()
    if err != nil {
        log.Fatalf("Failed to list partitions for device %s: %v", device, err)
    }

    // Если устройство уже смонтировано, выходим
    if strings.TrimSpace(string(output)) != "" {
        log.Printf("Device %s or its partitions are already mounted.", device)
        return
    }

    // Если нет разделов, монтируем само устройство
    if err := exec.Command("mount", device, mountPoint).Run(); err == nil {
        log.Printf("Successfully mounted %s to %s", device, mountPoint)
        return
    }

    // Если монтирование устройства не удалось, пробуем монтировать его разделы
    partDevice := fmt.Sprintf("%s1", device)
    if err := exec.Command("mount", partDevice, mountPoint).Run(); err != nil {
        log.Fatalf("Failed to mount device or partition %s: %v", partDevice, err)
    }
    log.Printf("Successfully mounted %s to %s", partDevice, mountPoint)
}

// Start запускает процесс мониторинга устройств
func Start() {
    err := CreateMediaDirectory()
    if err != nil {
        log.Fatalf("Failed to create /media directory: %v", err)
    }

    watcher, err := fsnotify.NewWatcher()
    if err != nil {
        log.Fatalf("Failed to create watcher: %v", err)
    }
    defer watcher.Close()

    done := make(chan bool)
    go func() {
        for {
            select {
            case event, ok := <-watcher.Events:
                if !ok {
                    return
                }
                if event.Op&fsnotify.Create == fsnotify.Create {
                    if strings.HasPrefix(event.Name, "/dev/sd") || strings.HasPrefix(event.Name, "/dev/nvme") {
                        log.Printf("Detected new device: %s", event.Name)
                        // Небольшая задержка для корректной инициализации устройства
                        time.Sleep(1 * time.Second)
                        mountDevice(event.Name)
                    }
                }
            case err, ok := <-watcher.Errors:
                if !ok {
                    return
                }
                log.Printf("Watcher error: %v", err)
            }
        }
    }()

    err = watcher.Add("/dev")
    if err != nil {
        log.Fatalf("Failed to add /dev to watcher: %v", err)
    }

    log.Println("Device monitoring started...")
    <-done
}
