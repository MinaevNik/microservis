package api

import (
    "encoding/json"
    "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
    "servis/pkg/update"
    "servis/pkg/shutdown" // Импорт пакета shutdown
    "servis/pkg/wifi"     // Импорт пакета wifi
    "github.com/gorilla/mux"
    "archive/zip"
)

var selectedZipFilePath string // переменная для хранения выбранного файла прошивки

// NetworkSelection представляет структуру для выбора Wi-Fi сети
type NetworkSelection struct {
    Name     string `json:"name"`
    Password string `json:"password"`
}

// ShutdownRequest представляет структуру для запроса на выключение устройства
type ShutdownRequest struct {
    Comment string `json:"comment"`
}

// RebootRequest представляет структуру для запроса на перезагрузку устройства
type RebootRequest struct {
    Comment string `json:"comment"`
}

// FileInfo содержит информацию о каждом файле внутри ZIP-файла
type FileInfo struct {
    Source      string `json:"source"`
    FileVersion string `json:"file_version"`
}

// ZipFileInfo содержит информацию о ZIP-файле и файлах с их версиями внутри
type ZipFileInfo struct {
    Path   string     `json:"path"`
    Files  []FileInfo `json:"files"`
}

// RegisterRoutes регистрирует маршруты для HTTP эндпоинтов.
func RegisterRoutes(r *mux.Router) {
    r.HandleFunc("/networks/all", GetNetworks).Methods("GET")
    r.HandleFunc("/networks/connect", ConnectNetwork).Methods("POST")
    r.HandleFunc("/shutdown", HandleShutdown).Methods("POST")
    r.HandleFunc("/reboot", HandleReboot).Methods("POST")
    r.HandleFunc("/usb/files", GetUSBFiles).Methods("GET")
    r.HandleFunc("/firmware/update", PerformFirmwareUpdate).Methods("POST")
    r.HandleFunc("/firmware/rollback", RollbackFirmwareHandler).Methods("POST")
}

// GetNetworks обрабатывает запрос на получение списка доступных сетей.
func GetNetworks(w http.ResponseWriter, r *http.Request) {
    networks, err := wifi.ScanNetworks("wlan0")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(networks)
}

// ConnectNetwork обрабатывает запрос на подключение к выбранной сети.
func ConnectNetwork(w http.ResponseWriter, r *http.Request) {
    var selection NetworkSelection
    if err := json.NewDecoder(r.Body).Decode(&selection); err != nil {
        http.Error(w, "invalid request payload", http.StatusBadRequest)
        return
    }

    err := wifi.UpdateNetworkConfig("/etc/wpa_supplicant/wpa_supplicant.conf", selection.Name, selection.Password)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    wifi.StopWpaSupplicant()

    _, err = wifi.RunCommand("ip", "link", "set", "wlan0", "down")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    _, err = wifi.RunCommand("ip", "link", "set", "wlan0", "up")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    _, err = wifi.RunCommand("wpa_supplicant", "-B", "-i", "wlan0", "-c", "/etc/wpa_supplicant/wpa_supplicant.conf")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    time.Sleep(5 * time.Second)

    _, err = wifi.RunCommand("dhclient", "wlan0")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("connected to network"))
}

// HandleShutdown обрабатывает запрос на выключение устройства.
func HandleShutdown(w http.ResponseWriter, r *http.Request) {
    var shutdownReq ShutdownRequest
    if err := json.NewDecoder(r.Body).Decode(&shutdownReq); err != nil {
        http.Error(w, "invalid request payload", http.StatusBadRequest)
        return
    }

    if shutdownReq.Comment == "shutdown now" {
        err := shutdown.Shutdown()
        if err != nil {
            http.Error(w, "failed to shutdown the system", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusOK)
        w.Write([]byte("system is shutting down"))
        return
    }

    http.Error(w, "invalid comment", http.StatusBadRequest)
}

// HandleReboot обрабатывает запрос на перезагрузку устройства.
func HandleReboot(w http.ResponseWriter, r *http.Request) {
    var rebootReq RebootRequest
    if err := json.NewDecoder(r.Body).Decode(&rebootReq); err != nil {
        http.Error(w, "invalid request payload", http.StatusBadRequest)
        return
    }

    if rebootReq.Comment == "reboot now" {
        err := shutdown.Reboot()
        if err != nil {
            http.Error(w, "failed to reboot the system", http.StatusInternalServerError)
            return
        }

        w.WriteHeader(http.StatusOK)
        w.Write([]byte("system is rebooting"))
        return
    }

    http.Error(w, "invalid comment", http.StatusBadRequest)
}

// GetUSBFiles возвращает список ZIP-файлов на USB-устройствах с информацией о файлах и их версиях.
func GetUSBFiles(w http.ResponseWriter, r *http.Request) {
    usbDevices, err := update.GetUSBMountPoints()
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to get USB devices: %v", err), http.StatusInternalServerError)
        return
    }

    var zipFilesInfo []ZipFileInfo
    for _, usbPath := range usbDevices {
        files, err := os.ReadDir(usbPath)
        if err != nil {
            http.Error(w, fmt.Sprintf("failed to read directory %s: %v", usbPath, err), http.StatusInternalServerError)
            return
        }

        for _, file := range files {
            if file.Type().IsRegular() && strings.HasSuffix(file.Name(), ".zip") {
                zipFilePath := usbPath + "/" + file.Name()
                fileInfos, err := extractFilesInfoFromZip(zipFilePath)
                if err != nil {
                    // Если возникла ошибка при извлечении информации, пропустим этот файл
                    log.Printf("failed to extract file info from %s: %v", zipFilePath, err)
                    continue
                }
                zipFilesInfo = append(zipFilesInfo, ZipFileInfo{
                    Path:  zipFilePath,
                    Files: fileInfos,
                })
            }
        }
    }

    if len(zipFilesInfo) == 0 {
        http.Error(w, "no zip files found", http.StatusNotFound)
        return
    }

    w.Header().Set("Content-Type", "application/json")
    json.NewEncoder(w).Encode(zipFilesInfo)
}

// extractFilesInfoFromZip извлекает информацию о файлах из JSON внутри ZIP-файла
func extractFilesInfoFromZip(zipFilePath string) ([]FileInfo, error) {
    zipReader, err := zip.OpenReader(zipFilePath)
    if err != nil {
        return nil, fmt.Errorf("failed to open zip file: %w", err)
    }
    defer zipReader.Close()

    firmwareInfo, err := update.FindValidFirmware(&zipReader.Reader)
    if err != nil {
        return nil, fmt.Errorf("failed to find valid firmware in zip file: %w", err)
    }

    var fileInfos []FileInfo
    for _, file := range firmwareInfo.Files {
        fileInfos = append(fileInfos, FileInfo{
            Source:      file.Source,
            FileVersion: file.FileVersion,
        })
    }

    return fileInfos, nil
}

// PerformFirmwareUpdate обрабатывает запрос на выполнение обновления прошивки.
func PerformFirmwareUpdate(w http.ResponseWriter, r *http.Request) {
    var req struct {
        SelectedFile string `json:"selected_file"`
    }
    if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
        http.Error(w, "invalid request payload", http.StatusBadRequest)
        return
    }

    if req.SelectedFile == "" {
        http.Error(w, "no file selected", http.StatusBadRequest)
        return
    }

    selectedZipFilePath = req.SelectedFile
    versionFilePath := "/root/dt_backend/installed_versions.json"
    backupDir := "/root/dt_backend/UpdateBackup"

    err := update.UpdateFirmware(selectedZipFilePath, versionFilePath, backupDir)
    if err != nil {
        http.Error(w, fmt.Sprintf("failed to update firmware: %v", err), http.StatusInternalServerError)
        return
    }
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Firmware update completed successfully"))
}

// RollbackFirmwareHandler обрабатывает запрос на откат прошивки.
func RollbackFirmwareHandler(w http.ResponseWriter, r *http.Request) {
    backupDir := "/root/dt_backend/UpdateBackup"

    versionFilePath := "/root/dt_backend/installed_versions.json"
    installedVersions, err := update.LoadInstalledVersions(versionFilePath)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to load installed versions: %v", err), http.StatusInternalServerError)
        return
    }

    err = update.RollbackFirmware(backupDir, installedVersions)
    if err != nil {
        http.Error(w, fmt.Sprintf("Failed to rollback firmware: %v", err), http.StatusInternalServerError)
        return
    }

    w.WriteHeader(http.StatusOK)
    w.Write([]byte("Firmware rollback completed successfully"))
}

// StartServer запускает HTTP сервер.
func StartServer() {
    r := mux.NewRouter()
    RegisterRoutes(r)

    log.Println("server is starting...")
    if err := http.ListenAndServe(":4444", r); err != nil {
        log.Fatalf("server failed to start: %v", err)
    }
}
