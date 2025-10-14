package main

import (
    "archive/tar"
    "compress/gzip"
    "fmt"
    "io"
    "net/http"
    "os"
    "os/exec"
    "path/filepath"
    "runtime"
    "strings"
)

const (
    pdArm64URL  = "https://github.com/jkawamoto/go-pixeldrain/releases/download/v0.7.5/pd_0.7.5_linux_arm64.tar.gz"
    pdAmd64URL  = "https://github.com/jkawamoto/go-pixeldrain/releases/download/v0.7.5/pd_0.7.5_linux_amd64.tar.gz"
    pdBinName   = "pd"
    pdTarName   = "pd.tar.gz"
)

func getArchURL() string {
    switch runtime.GOARCH {
    case "arm64":
        return pdArm64URL
    case "amd64":
        return pdAmd64URL
    default:
        fmt.Println("Unsupported architecture:", runtime.GOARCH)
        os.Exit(1)
        return ""
    }
}

func fileExists(filename string) bool {
    info, err := os.Stat(filename)
    if os.IsNotExist(err) {
        return false
    }
    return !info.IsDir()
}

func downloadPD(url string, dest string) error {
    resp, err := http.Get(url)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    out, err := os.Create(dest)
    if err != nil {
        return err
    }
    defer out.Close()
    _, err = io.Copy(out, resp.Body)
    return err
}

func extractTarGz(src string, target string) error {
    file, err := os.Open(src)
    if err != nil {
        return err
    }
    defer file.Close()
    gz, err := gzip.NewReader(file)
    if err != nil {
        return err
    }
    defer gz.Close()
    tr := tar.NewReader(gz)
    for {
        header, err := tr.Next()
        if err == io.EOF {
            break
        }
        if err != nil {
            return err
        }
        if header.Typeflag == tar.TypeReg && strings.HasSuffix(header.Name, pdBinName) {
            out, err := os.Create(target)
            if err != nil {
                return err
            }
            defer out.Close()
            _, err = io.Copy(out, tr)
            if err != nil {
                return err
            }
            os.Chmod(target, 0755)
            break
        }
    }
    return nil
}

func pdUploadMultiple(pdPath string, files []string) error {
    args := append([]string{"upload"}, files...)
    cmd := exec.Command(pdPath, args...)
    cmd.Stdout = os.Stdout
    cmd.Stderr = os.Stderr
    return cmd.Run()
}

func main() {
    if len(os.Args) != 2 {
        fmt.Println("Usage: uwu <device>")
        os.Exit(1)
    }

    if fileExists(pdBinName) {
        fmt.Println("pd binary found, skipping download.")
    } else {
        archURL := getArchURL()
        fmt.Println("Downloading pd for architecture:", runtime.GOARCH)
        if err := downloadPD(archURL, pdTarName); err != nil {
            fmt.Println("Download failed:", err)
            os.Exit(1)
        }
        fmt.Println("Extracting pd binary...")
        if err := extractTarGz(pdTarName, pdBinName); err != nil {
            fmt.Println("Extraction failed:", err)
            os.Exit(1)
        }
    }

    device := os.Args[1]
    baseDir := fmt.Sprintf("out/target/product/%s", device)

    patterns := []string{
        filepath.Join(baseDir, "*.zip"),
        filepath.Join(baseDir, "dtbo.img"),
        filepath.Join(baseDir, "vendor_boot.img"),
        filepath.Join(baseDir, "boot.img"),
    }

    var uploadFiles []string
    for _, pattern := range patterns {
        matches, err := filepath.Glob(pattern)
        if err != nil {
            fmt.Println("Erro ao buscar arquivos com padrão:", pattern)
            continue
        }
        if len(matches) == 0 {
            fmt.Printf("Aviso: Nenhum arquivo encontrado para %s, pulando...\n", pattern)
            continue
        }
        uploadFiles = append(uploadFiles, matches...)
    }

    if len(uploadFiles) == 0 {
        fmt.Println("Nenhum arquivo para upload encontrado. Saindo.")
        os.Exit(0)
    }

    fmt.Println("Uploading files:", uploadFiles)
    if err := pdUploadMultiple("./"+pdBinName, uploadFiles); err != nil {
        fmt.Println("Upload failed:", err)
        os.Exit(1)
    }
    fmt.Println("Upload completed successfully!")
}
