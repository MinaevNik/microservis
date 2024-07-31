package api

import (
    "encoding/json"
    "log"
    "net/http"
    "time"

    "github.com/gorilla/mux"
    "servis/pkg/shutdown"
    "servis/pkg/wifi"
)

type NetworkSelection struct {
    Name     string `json:"name"`
    Password string `json:"password"`
}

type ShutdownRequest struct {
    Comment string `json:"comment"`
}

type RebootRequest struct {
    Comment string `json:"comment"`
}

// RegisterRoutes регистрирует маршруты для HTTP эндпоинтов.
func RegisterRoutes(r *mux.Router) {
    r.HandleFunc("/networks/all", GetNetworks).Methods("GET")
    r.HandleFunc("/networks/connect", ConnectNetwork).Methods("POST")
    r.HandleFunc("/shutdown", HandleShutdown).Methods("POST")
    r.HandleFunc("/reboot", HandleReboot).Methods("POST")
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

// StartServer запускает HTTP сервер.
func StartServer() {
    r := mux.NewRouter()
    RegisterRoutes(r)

    log.Println("server is starting...")
    if err := http.ListenAndServe(":4444", r); err != nil {
        log.Fatalf("server failed to start: %v", err)
    }
}
