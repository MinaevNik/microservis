package ethernet

import (
    "bytes"
    "fmt"
    "io/ioutil"
    "os/exec"
    "strings"
)

// RunCommand выполняет команду в shell
func RunCommand(name string, args ...string) (string, error) {
    cmd := exec.Command(name, args...)
    var out bytes.Buffer
    cmd.Stdout = &out
    cmd.Stderr = &out
    err := cmd.Run()
    return out.String(), err
}

// GetEthernetInfo получает информацию о конфигурации Ethernet
func GetEthernetInfo(interfaceName string) (string, string, string, string, error) {
    ipInfo, err := RunCommand("ip", "addr", "show", interfaceName)
    if err != nil {
        return "", "", "", "", err
    }

    routeInfo, err := RunCommand("ip", "route")
    if err != nil {
        return "", "", "", "", err
    }

    ipLines := strings.Split(ipInfo, "\n")
    var ipAddr, netmask string
    for _, line := range ipLines {
        line = strings.TrimSpace(line)
        if strings.HasPrefix(line, "inet ") {
            parts := strings.Fields(line)
            ipAddr = parts[1]
            if idx := strings.Index(ipAddr, "/"); idx != -1 {
                netmask = ipAddr[idx+1:]
                ipAddr = ipAddr[:idx]
            }
            break
        }
    }

    routeLines := strings.Split(routeInfo, "\n")
    var gateway string
    for _, line := range routeLines {
        if strings.HasPrefix(line, "default via ") {
            parts := strings.Fields(line)
            gateway = parts[2]
            break
        }
    }

    dnsInfo, err := ioutil.ReadFile("/etc/resolv.conf")
    if err != nil {
        return "", "", "", "", err
    }
    dnsLines := strings.Split(string(dnsInfo), "\n")
    var dnsServers []string
    for _, line := range dnsLines {
        if strings.HasPrefix(line, "nameserver ") {
            parts := strings.Fields(line)
            dnsServers = append(dnsServers, parts[1])
        }
    }

    return ipAddr, netmask, gateway, strings.Join(dnsServers, " "), nil
}

// UpdateEthernetConfig обновляет конфигурационный файл Ethernet и WiFi
func UpdateEthernetConfig(filePath, ipAddr, netmask, gateway, dns string) error {
    content, err := ioutil.ReadFile(filePath)
    if err != nil {
        return err
    }

    lines := strings.Split(string(content), "\n")
    var newLines []string
    insideEthernetBlock := false
    insideWifiBlock := false
    ethernetBlockExists := false
    wifiBlockExists := false

    for _, line := range lines {
        trimmedLine := strings.TrimSpace(line)
        if strings.HasPrefix(trimmedLine, "iface eth0 inet static") {
            insideEthernetBlock = true
            ethernetBlockExists = true
        } else if strings.HasPrefix(trimmedLine, "iface wlan0 inet dhcp") {
            insideWifiBlock = true
            wifiBlockExists = true
        }

        if insideEthernetBlock {
            if strings.HasPrefix(trimmedLine, "address") {
                newLines = append(newLines, "    address "+ipAddr)
                continue
            }
            if strings.HasPrefix(trimmedLine, "netmask") {
                newLines = append(newLines, "    netmask "+netmask)
                continue
            }
            if strings.HasPrefix(trimmedLine, "gateway") {
                newLines = append(newLines, "    gateway "+gateway)
                continue
            }
            if strings.HasPrefix(trimmedLine, "dns-nameservers") {
                newLines = append(newLines, "    dns-nameservers "+dns)
                continue
            }
            if strings.HasPrefix(trimmedLine, "metric") {
                newLines = append(newLines, "    metric 100")
                continue
            }
            if trimmedLine == "" {
                if !strings.Contains(strings.Join(newLines, "\n"), "metric 100") {
                    newLines = append(newLines, "    metric 100")
                }
                insideEthernetBlock = false
            }
        } else if insideWifiBlock {
            if strings.HasPrefix(trimmedLine, "metric") {
                newLines = append(newLines, "    metric 200")
                continue
            }
            if trimmedLine == "" {
                if !strings.Contains(strings.Join(newLines, "\n"), "metric 200") {
                    newLines = append(newLines, "    metric 200")
                }
                insideWifiBlock = false
            }
        }

        newLines = append(newLines, line)
    }

    if !ethernetBlockExists {
        newEthernetBlock := fmt.Sprintf(`
# Ethernet
allow-hotplug eth0
iface eth0 inet static
    address %s
    netmask %s
    gateway %s
    dns-nameservers %s
    metric 100
`, ipAddr, netmask, gateway, dns)
        newLines = append(newLines, newEthernetBlock)
    }

    if !wifiBlockExists {
        newWifiBlock := `
# WiFi
allow-hotplug wlan0
iface wlan0 inet dhcp
    metric 200
wpa-conf /etc/wpa_supplicant/wpa_supplicant.conf
`
        newLines = append(newLines, newWifiBlock)
    }

    if err := ioutil.WriteFile(filePath, []byte(strings.Join(newLines, "\n")), 0644); err != nil {
        return err
    }

    return nil
}

// ConfigureEthernet выполняет настройку Ethernet и WiFi
func ConfigureEthernet() error {
    ipAddr, netmask, gateway, dns, err := GetEthernetInfo("eth0")
    if err != nil {
        return fmt.Errorf("error getting ethernet info: %w", err)
    }

    configFilePath := "/etc/network/interfaces"  // Путь к конфигурационному файлу Ethernet

    // Вызов функции для обновления конфигурации Ethernet и WiFi
    err = UpdateEthernetConfig(configFilePath, ipAddr, netmask, gateway, dns)
    if err != nil {
        return fmt.Errorf("error updating ethernet config: %w", err)
    }

    return nil
}
