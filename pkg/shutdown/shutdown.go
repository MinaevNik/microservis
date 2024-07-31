package shutdown

import (
    "fmt"
    "os/exec"
)

// Shutdown выключает устройство
func Shutdown() error {
    cmd := exec.Command("sudo", "shutdown", "now")
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to shutdown the system: %w", err)
    }
    return nil
}

// Reboot перезагружает устройство
func Reboot() error {
    cmd := exec.Command("sudo", "reboot")
    err := cmd.Run()
    if err != nil {
        return fmt.Errorf("failed to reboot the system: %w", err)
    }
    return nil
}
