package device

import (
	"fmt"
	"log"
	"os/exec"
	"strings"
	"time"
)

// CreateMediaDirectory создаёт директорию /media, если она отсутствует
func CreateMediaDirectory() error {
	cmd := exec.Command("mkdir", "-p", "/media")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to create /media directory: %w", err)
	}
	return nil
}

// CheckAndMountDevices проверяет устройства и монтирует их, если они съёмные и не смонтированные
func CheckAndMountDevices() error {
	cmd := exec.Command("lsblk", "-o", "NAME,RM,SIZE,RO,TYPE,MOUNTPOINT")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to execute lsblk: %w", err)
	}
	fmt.Printf("lsblk output:\n%s\n", output)

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 5 {
			log.Printf("Unexpected output format: %v", line)
			continue
		}

		name := cleanDeviceName(fields[0])
		rm := fields[1]
		typ := fields[4]
		mountpoint := ""
		if len(fields) >= 6 {
			mountpoint = fields[5]
		}

		if rm == "1" && typ == "part" && mountpoint == "" {
			fmt.Printf("Found removable device: /dev/%s\n", name)
			mountDevice(name)
		} else {
			log.Printf("Skipping non-removable or already mounted device: %v", line)
		}
	}
	return nil
}

func cleanDeviceName(name string) string {
	name = strings.TrimLeft(name, "│─└├")
	return name
}

func mountDevice(device string) {
	mountPoint := fmt.Sprintf("/media/%s", device)
	err := exec.Command("mkdir", "-p", mountPoint).Run()
	if err != nil {
		log.Fatalf("Failed to create mount point %s: %v", mountPoint, err)
	}

	devicePath := fmt.Sprintf("/dev/%s", device)
	fmt.Printf("Attempting to mount %s to %s\n", devicePath, mountPoint)
	cmd := exec.Command("mount", devicePath, mountPoint)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("Failed to mount device %s: %v\n%s", devicePath, err, output)
	}

	fmt.Printf("Device %s mounted at %s\n", devicePath, mountPoint)
}

// StartDeviceMonitor запускает мониторинг устройств и их автомонтирование
func StartDeviceMonitor() {
	err := CreateMediaDirectory()
	if err != nil {
		log.Fatalf("Failed to create /media directory: %v", err)
	}

	for {
		err := CheckAndMountDevices()
		if err != nil {
			log.Printf("Error checking and mounting devices: %v", err)
		}
		time.Sleep(10 * time.Second) // Периодичность проверки устройств
	}
}