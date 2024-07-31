package wifi

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
	"time"
)

func RunCommand(name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return out.String(), err
}

func StopWpaSupplicant() {
	pids, err := RunCommand("pgrep", "wpa_supplicant")
	if err != nil {
		fmt.Printf("error getting wpa_supplicant PID: %v\n", err)
		return
	}

	for _, pid := range strings.Split(strings.TrimSpace(pids), "\n") {
		if pid != "" {
			RunCommand("kill", "-9", pid)
		}
	}

	time.Sleep(2 * time.Second) // Adding delay for process termination
}

func UpdateNetworkConfig(filePath, ssid, psk string) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	insideNetworkBlock := false
	networkBlockExists := false

	for _, line := range lines {
		trimmedLine := strings.TrimSpace(line)
		if strings.HasPrefix(trimmedLine, "network={") {
			insideNetworkBlock = true
			networkBlockExists = true
		}

		if insideNetworkBlock && strings.HasPrefix(trimmedLine, "ssid=") {
			newLines = append(newLines, `    ssid="`+ssid+`"`)
			continue
		}

		if insideNetworkBlock && strings.HasPrefix(trimmedLine, "psk=") {
			newLines = append(newLines, `    psk="`+psk+`"`)
			continue
		}

		if insideNetworkBlock && trimmedLine == "}" {
			insideNetworkBlock = false
		}

		newLines = append(newLines, line)
	}

	if !networkBlockExists {
		// Adding a new network block if it doesn't exist
		newNetworkBlock := fmt.Sprintf(`
network={
    ssid="%s"
    psk="%s"
    key_mgmt=WPA-PSK
}`, ssid, psk)
		newLines = append(newLines, newNetworkBlock)
	}

	if err := ioutil.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func ScanNetworks(wifiInterface string) ([]map[string]interface{}, error) {
	output, err := RunCommand("iwlist", wifiInterface, "scan")
	if err != nil {
		return nil, fmt.Errorf("error scanning networks: %v\nOutput: %s", err, output)
	}

	lines := strings.Split(output, "\n")
	var networks []map[string]interface{}
	var currentNetwork map[string]interface{}

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Cell") {
			if currentNetwork != nil {
				networks = append(networks, currentNetwork)
			}
			currentNetwork = make(map[string]interface{})
		}

		if strings.Contains(line, "ESSID:") {
			ssid := strings.TrimPrefix(line, "ESSID:")
			ssid = strings.Trim(ssid, "\"")
			currentNetwork["name"] = ssid
		}

		if strings.Contains(line, "Quality=") {
			quality := strings.Split(strings.Split(line, "=")[1], "/")[0]
			currentNetwork["quality"] = quality
		}
	}

	if currentNetwork != nil {
		networks = append(networks, currentNetwork)
	}

	return networks, nil
}
