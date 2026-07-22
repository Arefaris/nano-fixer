package localai

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"
	"time"
)

var (
	engineCmd   *exec.Cmd
	engineMutex sync.Mutex
)

// StartEngine starts the llama-server process silently in the background
func StartEngine() error {
	engineMutex.Lock()
	defer engineMutex.Unlock()

	if engineCmd != nil && engineCmd.Process != nil {
		return nil // Already running
	}

	exeDir, _ := os.Executable()
	localAIDir := filepath.Join(filepath.Dir(exeDir), "local_ai")
	serverPath := filepath.Join(localAIDir, "llama-server.exe")
	modelPath := filepath.Join(localAIDir, ModelFilename)

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return fmt.Errorf("engine not found")
	}

	cmd := exec.Command(serverPath, "--model", modelPath, "--port", "8080", "--ctx-size", "2048", "--parallel", "1", "-cb")
	// Hide window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// We can redirect output to a log file if needed
	logPath := filepath.Join(localAIDir, "engine.log")
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0666)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	engineCmd = cmd
	log.Println("Local AI engine started successfully (PID:", cmd.Process.Pid, ")")

	// Wait for the server to be ready
	go func() {
		for i := 0; i < 30; i++ {
			time.Sleep(1 * time.Second)
			resp, err := http.Get("http://127.0.0.1:8080/v1/models")
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				log.Println("Local AI engine is ready to receive requests.")
				return
			}
		}
		log.Println("Local AI engine took too long to start or failed.")
	}()

	return nil
}

// StopEngine stops the running llama-server process
func StopEngine() {
	engineMutex.Lock()
	defer engineMutex.Unlock()

	if engineCmd != nil && engineCmd.Process != nil {
		log.Println("Stopping Local AI engine...")
		engineCmd.Process.Kill()
		engineCmd.Wait()
		engineCmd = nil
		log.Println("Local AI engine stopped.")
	}
}

// IsRunning returns true if the engine is currently running
func IsRunning() bool {
	engineMutex.Lock()
	defer engineMutex.Unlock()
	return engineCmd != nil && engineCmd.Process != nil
}
