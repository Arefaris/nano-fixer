package localai

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	downloadStatus string
	downloadPct    int
	downloadMutex  sync.RWMutex
	isDownloading  bool

	LlamaServerURL = "https://github.com/ggerganov/llama.cpp/releases/download/b4604/llama-b4604-bin-win-vulkan-x64.zip"
	ModelURL       = "https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf?download=true"
	ModelFilename  = "qwen2.5-0.5b-instruct-q4_k_m.gguf"
)

func GetProgress() (int, string, bool) {
	downloadMutex.RLock()
	defer downloadMutex.RUnlock()
	return downloadPct, downloadStatus, isDownloading
}

func setProgress(pct int, status string) {
	downloadMutex.Lock()
	downloadPct = pct
	downloadStatus = status
	downloadMutex.Unlock()
}

type progressWriter struct {
	total      int64
	downloaded int64
	onProgress func(int64, int64)
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.downloaded += int64(n)
	pw.onProgress(pw.downloaded, pw.total)
	return n, nil
}

// CheckFilesExist checks if the engine and model already exist.
func CheckFilesExist() bool {
	exeDir, _ := os.Executable()
	localAIDir := filepath.Clean(filepath.Join(filepath.Dir(exeDir), "local_ai"))
	serverPath := filepath.Clean(filepath.Join(localAIDir, "llama-server.exe"))
	modelPath := filepath.Clean(filepath.Join(localAIDir, ModelFilename))

	if !strings.HasPrefix(serverPath, localAIDir) || !strings.HasPrefix(modelPath, localAIDir) {
		return false
	}

	_, err1 := os.Stat(serverPath)
	_, err2 := os.Stat(modelPath)
	return err1 == nil && err2 == nil
}

func EnsureLocalAIFiles() error {
	downloadMutex.Lock()
	if isDownloading {
		downloadMutex.Unlock()
		return fmt.Errorf("already downloading")
	}
	isDownloading = true
	downloadMutex.Unlock()

	defer func() {
		downloadMutex.Lock()
		isDownloading = false
		downloadMutex.Unlock()
	}()

	exeDir, _ := os.Executable()
	localAIDir := filepath.Clean(filepath.Join(filepath.Dir(exeDir), "local_ai"))
	os.MkdirAll(localAIDir, 0750) // More restrictive permissions

	serverPath := filepath.Clean(filepath.Join(localAIDir, "llama-server.exe"))
	modelPath := filepath.Clean(filepath.Join(localAIDir, ModelFilename))

	if !strings.HasPrefix(serverPath, localAIDir) || !strings.HasPrefix(modelPath, localAIDir) {
		return fmt.Errorf("invalid path traversal attempt")
	}

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		err := downloadAndExtractZip(LlamaServerURL, localAIDir, "llama-server.exe")
		if err != nil {
			setProgress(0, "Error downloading engine: "+err.Error())
			return err
		}
	}

	if _, err := os.Stat(modelPath); os.IsNotExist(err) {
		err := downloadFile(ModelURL, modelPath, "Downloading AI Model (350MB)...")
		if err != nil {
			setProgress(0, "Error downloading model: "+err.Error())
			return err
		}
	}

	setProgress(100, "Ready.")
	return nil
}

func downloadFile(url, dest, statusMsg string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	total := resp.ContentLength
	if total > 5*1024*1024*1024 { // 5 GB limit for model
		return fmt.Errorf("file too large")
	}

	tmpDest := dest + ".tmp"
	out, err := os.OpenFile(tmpDest, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0640)
	if err != nil {
		return err
	}

	pw := &progressWriter{
		total: total,
		onProgress: func(downloaded, total int64) {
			if total > 0 {
				pct := int(float64(downloaded) / float64(total) * 100)
				setProgress(pct, statusMsg)
			} else {
				setProgress(0, fmt.Sprintf("%s (downloading...)", statusMsg))
			}
		},
	}

	// Limit reader to prevent disk exhaustion
	_, err = io.Copy(out, io.TeeReader(io.LimitReader(resp.Body, 5*1024*1024*1024), pw))
	out.Close()
	if err != nil {
		os.Remove(tmpDest)
		return err
	}

	return os.Rename(tmpDest, dest)
}

func downloadAndExtractZip(url, destDir, targetFile string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	total := resp.ContentLength
	if total > 500*1024*1024 { // 500 MB limit
		return fmt.Errorf("file too large")
	}

	var buf bytes.Buffer
	pw := &progressWriter{
		total: total,
		onProgress: func(downloaded, total int64) {
			if total > 0 {
				pct := int(float64(downloaded) / float64(total) * 100)
				setProgress(pct, "Downloading Engine...")
			}
		},
	}

	// Read with a hard limit to prevent memory exhaustion
	limitReader := io.LimitReader(resp.Body, 500*1024*1024)
	_, err = io.Copy(&buf, io.TeeReader(limitReader, pw))
	if err != nil {
		return err
	}

	setProgress(100, "Extracting Engine...")

	reader, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return err
	}

	for _, file := range reader.File {
		// Clean the file name to prevent directory traversal in archive (Zip Slip)
		cleanName := filepath.Clean(file.Name)
		if strings.Contains(cleanName, "..") {
			continue // Skip malicious files
		}

		if strings.HasSuffix(cleanName, targetFile) {
			f, err := file.Open()
			if err != nil {
				return err
			}
			destPath := filepath.Join(destDir, targetFile)
			// Final check
			if !strings.HasPrefix(destPath, filepath.Clean(destDir)) {
				f.Close()
				continue
			}
			out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0750)
			if err != nil {
				f.Close()
				return err
			}
			// Limit extraction size to prevent zip bombs
			_, err = io.Copy(out, io.LimitReader(f, 100*1024*1024))
			f.Close()
			out.Close()
			if err != nil {
				return err
			}
			break
		}
	}

	// Extract .dll files as well safely
	for _, file := range reader.File {
		cleanName := filepath.Clean(file.Name)
		if strings.Contains(cleanName, "..") {
			continue
		}

		if strings.HasSuffix(cleanName, ".dll") {
			f, err := file.Open()
			if err == nil {
				destPath := filepath.Join(destDir, filepath.Base(cleanName))
				if strings.HasPrefix(destPath, filepath.Clean(destDir)) {
					out, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0750)
					if err == nil {
						io.Copy(out, io.LimitReader(f, 50*1024*1024))
						out.Close()
					}
				}
				f.Close()
			}
		}
	}

	return nil
}
