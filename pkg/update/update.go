package update

import (
    "archive/zip"
    "crypto/sha256"
    "encoding/hex"
    "encoding/json"
    "fmt"
    "io"
    "io/ioutil"
    "log"
    "os"
    "os/exec"
    "path/filepath"
    "regexp"
    "sort"
    "strings"
)

// FirmwareInfo содержит информацию из JSON-файла о прошивке
type FirmwareInfo struct {
    Files []struct {
        Source      string `json:"source"`
        Destination string `json:"destination"`
        FileVersion string `json:"file_version"`
        IsDir       bool   `json:"is_dir"`
        Hash        string `json:"hash"`
    } `json:"files"`
}

// InstalledVersionInfo содержит информацию о текущих версиях установленных файлов и директорий
type InstalledVersionInfo struct {
    Files []struct {
        Destination string `json:"destination"`
        FileVersion string `json:"file_version"`
    } `json:"files"`
}

// GetUSBMountPoints возвращает список всех смонтированных USB-устройств.
func GetUSBMountPoints() ([]string, error) {
    cmd := exec.Command("lsblk", "-o", "MOUNTPOINT,TYPE")
    var out strings.Builder
    cmd.Stdout = &out
    err := cmd.Run()
    if err != nil {
        return nil, fmt.Errorf("failed to run lsblk command: %w", err)
    }

    var usbMountPoints []string
    lines := strings.Split(out.String(), "\n")
    for _, line := range lines {
        if strings.Contains(line, "part") {
            fields := strings.Fields(line)
            if len(fields) > 0 && fields[0] != "" {
                usbMountPoints = append(usbMountPoints, fields[0])
            }
        }
    }

    if len(usbMountPoints) == 0 {
        return nil, fmt.Errorf("no USB mount points found")
    }

    return usbMountPoints, nil
}

// FindValidFirmware извлекает информацию о прошивке из JSON-файла в ZIP-архиве
func FindValidFirmware(zipReader *zip.Reader) (*FirmwareInfo, error) {
    var firmwareInfo *FirmwareInfo

    for _, file := range zipReader.File {
        if strings.HasSuffix(file.Name, ".json") {
            f, err := file.Open()
            if err != nil {
                return nil, fmt.Errorf("failed to open JSON file in zip: %w", err)
            }
            defer f.Close()

            jsonContent, err := ioutil.ReadAll(f)
            if err != nil {
                return nil, fmt.Errorf("failed to read JSON file: %w", err)
            }

            firmwareInfo = &FirmwareInfo{}
            err = json.Unmarshal(jsonContent, firmwareInfo)
            if err != nil {
                return nil, fmt.Errorf("failed to unmarshal JSON content: %w", err)
            }
            break
        }
    }

    if firmwareInfo == nil {
        return nil, fmt.Errorf("valid JSON file not found")
    }

    return firmwareInfo, nil
}

// calculateHash вычисляет SHA-256 хеш для данных
func calculateHash(data []byte) string {
    hasher := sha256.New()
    hasher.Write(data)
    return hex.EncodeToString(hasher.Sum(nil))
}

// calculateFileHash вычисляет хеш для файла
func calculateFileHash(filePath string) (string, error) {
    data, err := ioutil.ReadFile(filePath)
    if err != nil {
        return "", err
    }
    return calculateHash(data), nil
}

// calculateDirectoryHash вычисляет хеш директории путем объединения хешей всех файлов в этой директории и поддиректориях
func calculateDirectoryHash(dirPath string) (string, error) {
    var fileHashes []string

    err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() && path != dirPath {
            return filepath.SkipDir
        }
        if !info.IsDir() {
            fileHash, err := calculateFileHash(path)
            if err != nil {
                return err
            }
            fileHashes = append(fileHashes, fileHash)
        }
        return nil
    })

    if err != nil {
        return "", err
    }

    sort.Strings(fileHashes)
    combinedHashes := strings.Join(fileHashes, "")
    return calculateHash([]byte(combinedHashes)), nil
}

// createBackup создает резервную копию файла или директории
func createBackup(source, backupDir string) error {
    backupPath := filepath.Join(backupDir, filepath.Base(source))

    info, err := os.Stat(source)
    if err != nil {
        return fmt.Errorf("failed to stat source for backup: %w", err)
    }

    if info.IsDir() {
        return copyDirectory(source, backupPath)
    } else {
        return copyFile(source, backupPath)
    }
}

// copyFile копирует файл
func copyFile(source, destination string) error {
    input, err := ioutil.ReadFile(source)
    if err != nil {
        return fmt.Errorf("failed to read source file: %w", err)
    }

    err = os.MkdirAll(filepath.Dir(destination), 0755)
    if err != nil {
        return fmt.Errorf("failed to create destination directory: %w", err)
    }

    err = ioutil.WriteFile(destination, input, 0644)
    if err != nil {
        return fmt.Errorf("failed to write to destination file: %w", err)
    }

    fmt.Printf("Backup created for file %s at %s\n", source, destination)
    return nil
}

// copyDirectory копирует директорию
func copyDirectory(source, destination string) error {
    err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }

        relPath, err := filepath.Rel(source, path)
        if err != nil {
            return err
        }
        destPath := filepath.Join(destination, relPath)

        if info.IsDir() {
            err := os.MkdirAll(destPath, 0755)
            if err != nil {
                return fmt.Errorf("failed to create directory: %w", err)
            }
        } else {
            err := copyFile(path, destPath)
            if err != nil {
                return err
            }
        }

        return nil
    })

    if err != nil {
        return fmt.Errorf("failed to copy directory: %w", err)
    }

    fmt.Printf("Backup created for directory %s at %s\n", source, destination)
    return nil
}

// restoreBackup восстанавливает файлы и директории из бэкапа
func restoreBackup(backupDir string, installedVersions *InstalledVersionInfo) error {
    for _, file := range installedVersions.Files {
        backupPath := filepath.Join(backupDir, filepath.Base(file.Destination))

        info, err := os.Stat(backupPath)
        if err != nil {
            return fmt.Errorf("failed to stat backup: %w", err)
        }

        if info.IsDir() {
            err := copyDirectory(backupPath, file.Destination)
            if err != nil {
                return fmt.Errorf("failed to restore directory from backup: %w", err)
            }
        } else {
            err := copyFile(backupPath, file.Destination)
            if err != nil {
                return fmt.Errorf("failed to restore file from backup: %w", err)
            }
        }
    }

    fmt.Println("Backup restored successfully")
    return nil
}

// LoadInstalledVersions загружает информацию о текущих версиях установленных файлов
func LoadInstalledVersions(versionFilePath string) (*InstalledVersionInfo, error) {
    var installedVersionInfo InstalledVersionInfo

    if _, err := os.Stat(versionFilePath); os.IsNotExist(err) {
        err = ioutil.WriteFile(versionFilePath, []byte(`{"files": []}`), 0644)
        if err != nil {
            return nil, fmt.Errorf("failed to create initial version file: %w", err)
        }
    }

    data, err := ioutil.ReadFile(versionFilePath)
    if err != nil {
        return nil, fmt.Errorf("failed to read version file: %w", err)
    }

    err = json.Unmarshal(data, &installedVersionInfo)
    if err != nil {
        return nil, fmt.Errorf("failed to unmarshal version file: %w", err)
    }

    return &installedVersionInfo, nil
}

// saveInstalledVersions сохраняет информацию о текущих версиях установленных файлов
func saveInstalledVersions(versionFilePath string, installedVersionInfo *InstalledVersionInfo) error {
    data, err := json.MarshalIndent(installedVersionInfo, "", "  ")
    if err != nil {
        return fmt.Errorf("failed to marshal version info: %w", err)
    }

    err = ioutil.WriteFile(versionFilePath, data, 0644)
    if err != nil {
        return fmt.Errorf("failed to write version file: %w", err)
    }

    return nil
}

// compareVersions сравнивает версии (возвращает true, если v1 >= v2)
func compareVersions(v1, v2 string) (bool, error) {
    re := regexp.MustCompile(`^(\d+)\.(\d+)\.(\d+)$`)
    m1 := re.FindStringSubmatch(v1)
    m2 := re.FindStringSubmatch(v2)

    if len(m1) < 4 || len(m2) < 4 {
        return false, fmt.Errorf("invalid version format")
    }

    for i := 1; i < 4; i++ {
        if m1[i] > m2[i] {
            return true, nil
        } else if m1[i] < m2[i] {
            return false, nil
        }
    }
    return true, nil
}

// copyFileFromZipWithBackupAndChecks проверяет и копирует файл из zip в указанное место с созданием резервной копии, проверкой версии и хеша
func copyFileFromZipWithBackupAndChecks(zipReader *zip.ReadCloser, source, destination, backupDir, newVersion, expectedHash string, installedVersions *InstalledVersionInfo) error {
    var currentVersion string
    for _, file := range installedVersions.Files {
        if file.Destination == destination {
            currentVersion = file.FileVersion
            break
        }
    }

    if currentVersion != "" {
        isNewer, err := compareVersions(newVersion, currentVersion)
        if err != nil {
            return fmt.Errorf("failed to compare versions: %w", err)
        }
        if !isNewer {
            fmt.Printf("Skipping file %s: current version %s is newer or equal to %s\n", destination, currentVersion, newVersion)
            return nil
        }

        // Проверяем хеш, если версия новее
        actualHash, err := calculateFileHash(destination)
        if err != nil {
            return fmt.Errorf("failed to calculate current file hash: %w", err)
        }
        if actualHash != expectedHash {
            return fmt.Errorf("hash mismatch for file %s: expected %s, got %s", destination, expectedHash, actualHash)
        }
    }

    err := createBackup(destination, backupDir)
    if err != nil {
        return fmt.Errorf("failed to create backup for %s: %w", destination, err)
    }

    for _, file := range zipReader.File {
        if file.Name == source {
            fmt.Println("Copying file from zip:", file.Name)
            srcFile, err := file.Open()
            if err != nil {
                return fmt.Errorf("failed to open source file in zip: %w", err)
            }
            defer srcFile.Close()

            destDir := filepath.Dir(destination)
            err = os.MkdirAll(destDir, 0755)
            if err != nil {
                return fmt.Errorf("failed to create destination directory: %w", err)
            }

            destFile, err := os.Create(destination)
            if err != nil {
                return fmt.Errorf("failed to create destination file: %w", err)
            }
            defer destFile.Close()

            _, err = io.Copy(destFile, srcFile)
            if err != nil {
                return fmt.Errorf("failed to copy file content: %w", err)
            }

            // Обновляем версию файла в installedVersions
            updated := false
            for i := range installedVersions.Files {
                if installedVersions.Files[i].Destination == destination {
                    installedVersions.Files[i].FileVersion = newVersion
                    updated = true
                    break
                }
            }
            if !updated {
                installedVersions.Files = append(installedVersions.Files, struct {
                    Destination string `json:"destination"`
                    FileVersion string `json:"file_version"`
                }{Destination: destination, FileVersion: newVersion})
            }

            return nil
        }
    }
    return fmt.Errorf("file %s not found in zip", source)
}

// copyDirectoryFromZipWithBackupAndChecks проверяет и копирует директорию из zip в указанное место с созданием резервной копии, проверкой версии и хеша
func copyDirectoryFromZipWithBackupAndChecks(zipReader *zip.ReadCloser, source, destination, backupDir, newVersion, expectedHash string, installedVersions *InstalledVersionInfo) error {
    var currentVersion string
    for _, file := range installedVersions.Files {
        if file.Destination == destination {
            currentVersion = file.FileVersion
            break
        }
    }

    if currentVersion != "" {
        isNewer, err := compareVersions(newVersion, currentVersion)
        if err != nil {
            return fmt.Errorf("failed to compare versions: %w", err)
        }
        if !isNewer {
            fmt.Printf("Skipping directory %s: current version %s is newer or equal to %s\n", destination, currentVersion, newVersion)
            return nil
        }

        // Проверяем хеш, если версия новее
        actualHash, err := calculateDirectoryHash(destination)
        if err != nil {
            return fmt.Errorf("failed to calculate current directory hash: %w", err)
        }
        if actualHash != expectedHash {
            return fmt.Errorf("hash mismatch for directory %s: expected %s, got %s", destination, expectedHash, actualHash)
        }
    }

    err := createBackup(destination, backupDir)
    if err != nil {
        return fmt.Errorf("failed to create backup for %s: %w", destination, err)
    }

    for _, file := range zipReader.File {
        if strings.HasPrefix(file.Name, source) {
            relativePath := strings.TrimPrefix(file.Name, source)
            destPath := filepath.Join(destination, relativePath)
            if file.FileInfo().IsDir() {
                fmt.Println("Creating directory from zip:", destPath)
                err := os.MkdirAll(destPath, 0755)
                if err != nil {
                    return fmt.Errorf("failed to create directory: %w", err)
                }
            } else {
                fmt.Println("Copying file from zip:", file.Name)
                srcFile, err := file.Open()
                if err != nil {
                    return fmt.Errorf("failed to open source file in zip: %w", err)
                }
                defer srcFile.Close()

                destFile, err := os.Create(destPath)
                if err != nil {
                    return fmt.Errorf("failed to create destination file: %w", err)
                }
                defer destFile.Close()

                _, err = io.Copy(destFile, srcFile)
                if err != nil {
                    return fmt.Errorf("failed to copy file content: %w", err)
                }
            }
        }
    }

    // Обновляем версию директории в installedVersions
    updated := false
    for i := range installedVersions.Files {
        if installedVersions.Files[i].Destination == destination {
            installedVersions.Files[i].FileVersion = newVersion
            updated = true
            break
        }
    }
    if !updated {
        installedVersions.Files = append(installedVersions.Files, struct {
            Destination string `json:"destination"`
            FileVersion string `json:"file_version"`
        }{Destination: destination, FileVersion: newVersion})
    }

    return nil
}

// UpdateFirmware выполняет основную функцию обновления прошивки
func UpdateFirmware(zipFilePath string, versionFilePath string, backupDir string) error {
    log.Printf("Starting firmware update with zip file: %s", zipFilePath)
    zipReader, err := zip.OpenReader(zipFilePath)
    if err != nil {
        return fmt.Errorf("failed to open zip file: %w", err)
    }
    defer zipReader.Close()

    firmwareInfo, err := FindValidFirmware(&zipReader.Reader)
    if err != nil {
        return fmt.Errorf("failed to find valid firmware: %w", err)
    }

    installedVersions, err := LoadInstalledVersions(versionFilePath)
    if err != nil {
        return fmt.Errorf("failed to load installed versions: %w", err)
    }

    for _, file := range firmwareInfo.Files {
        if file.IsDir {
            err := copyDirectoryFromZipWithBackupAndChecks(zipReader, file.Source, file.Destination, backupDir, file.FileVersion, file.Hash, installedVersions)
            if err != nil {
                return fmt.Errorf("failed to copy directory from zip: %w", err)
            }
            log.Printf("Updated or added directory %s to %s\n", file.Source, file.Destination)
        } else {
            err := copyFileFromZipWithBackupAndChecks(zipReader, file.Source, file.Destination, backupDir, file.FileVersion, file.Hash, installedVersions)
            if err != nil {
                return fmt.Errorf("failed to copy file from zip: %w", err)
            }
            log.Printf("Updated or added file %s to %s\n", file.Source, file.Destination)
        }
    }

    err = saveInstalledVersions(versionFilePath, installedVersions)
    if err != nil {
        return fmt.Errorf("failed to save installed versions: %w", err)
    }

    log.Println("Firmware update completed successfully")
    return nil
}

// RollbackFirmware выполняет основную функцию отката прошивки
func RollbackFirmware(backupDir string, installedVersions *InstalledVersionInfo) error {
    log.Println("Starting firmware rollback")
    return restoreBackup(backupDir, installedVersions)
}
