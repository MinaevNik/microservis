package rtc

import (
    "fmt"
    "io/ioutil"
    "os/exec"
    "strings"
)

// RunCommand выполняет команду в shell
func RunCommand(name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    output, err := cmd.CombinedOutput()
    return string(output), err
}

// EnableI2C включает интерфейс I2C
func EnableI2C() error {
    configFile := "/boot/config.txt"
    content, err := ioutil.ReadFile(configFile)
    if err != nil {
        return err
    }

    lines := strings.Split(string(content), "\n")
    var newLines []string
    for _, line := range lines {
        if line == "dtparam=i2c_arm=on" || line == "dtoverlay=i2c-rtc,ds3231" {
            return nil
        }
        newLines = append(newLines, line)
    }
    newLines = append(newLines, "dtparam=i2c_arm=on")
    newLines = append(newLines, "dtoverlay=i2c-rtc,ds3231")

    return ioutil.WriteFile(configFile, []byte(strings.Join(newLines, "\n")), 0644)
}

// RemoveFakeHwClock удаляет пакет fake-hwclock
func RemoveFakeHwClock() error {
    _, err := RunCommand("sudo", "apt", "-y", "remove", "fake-hwclock")
    if err != nil {
        return err
    }
    _, err = RunCommand("sudo", "update-rc.d", "-f", "fake-hwclock", "remove")
    return err
}

// UpdateHwClockSetScript обновляет скрипт hwclock-set
func UpdateHwClockSetScript() error {
    scriptFile := "/lib/udev/hwclock-set"
    content, err := ioutil.ReadFile(scriptFile)
    if err != nil {
        return err
    }

    lines := strings.Split(string(content), "\n")
    var newLines []string
    for _, line := range lines {
        if line == "if [ -e /run/systemd/system ] ; then" ||
            line == "    exit 0" ||
            line == "fi" {
            newLines = append(newLines, "#"+line)
        } else {
            newLines = append(newLines, line)
        }
    }

    return ioutil.WriteFile(scriptFile, []byte(strings.Join(newLines, "\n")), 0644)
}

// SyncTime синхронизирует время с RTC
func SyncTime() error {
    _, err := RunCommand("sudo", "hwclock", "-w")
    return err
}

// ConfigureRTC настраивает модуль RTC без перезагрузки
func ConfigureRTC() error {
    err := EnableI2C()
    if err != nil {
        return fmt.Errorf("failed to enable I2C: %w", err)
    }

    err = RemoveFakeHwClock()
    if err != nil {
        return fmt.Errorf("failed to remove fake-hwclock: %w", err)
    }

    err = UpdateHwClockSetScript()
    if err != nil {
        return fmt.Errorf("failed to update hwclock-set script: %w", err)
    }

    err = SyncTime()
    if err != nil {
        return fmt.Errorf("failed to sync time with RTC: %w", err)
    }

    return nil
}
