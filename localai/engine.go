package localai

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	localAIDir := filepath.Clean(filepath.Join(filepath.Dir(exeDir), "local_ai"))
	serverPath := filepath.Clean(filepath.Join(localAIDir, "llama-server.exe"))
	modelPath := filepath.Clean(filepath.Join(localAIDir, ModelFilename))

	// Ensure the constructed paths are actually inside localAIDir (prevent path traversal)
	if !strings.HasPrefix(serverPath, localAIDir) || !strings.HasPrefix(modelPath, localAIDir) {
		return fmt.Errorf("invalid path traversal attempt")
	}

	if _, err := os.Stat(serverPath); os.IsNotExist(err) {
		return fmt.Errorf("engine not found")
	}

	// Safe command execution: os/exec escapes arguments automatically.
	cmd := exec.Command(serverPath, "--model", modelPath, "--port", "8080", "--ctx-size", "2048", "--parallel", "1", "-cb")
	// Hide window on Windows
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	// We can redirect output to a log file if needed
	logPath := filepath.Clean(filepath.Join(localAIDir, "engine.log"))
	logFile, _ := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	cmd.Stdout = logFile
	cmd.Stderr = logFile

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start engine: %w", err)
	}

	engineCmd = cmd
	log.Println("Local AI engine started successfully (PID:", cmd.Process.Pid, ")")

	// Monitor process exit
	go func() {
		err := cmd.Wait()
		log.Printf("Local AI engine process exited: %v", err)
		logFile.Close()
		
		engineMutex.Lock()
		if engineCmd == cmd {
			engineCmd = nil
		}
		engineMutex.Unlock()
	}()

	// Wait for the server to be ready
	for i := 0; i < 30; i++ {
		time.Sleep(1 * time.Second)
		resp, err := http.Get("http://127.0.0.1:8080/v1/models")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			log.Println("Local AI engine is ready to receive requests.")
			return nil
		}
		
		// If process exited while we were waiting, abort early
		engineMutex.Lock()
		if engineCmd == nil {
			engineMutex.Unlock()
			return fmt.Errorf("engine crashed during startup")
		}
		engineMutex.Unlock()
	}
	
	return fmt.Errorf("Local AI engine took too long to start or failed.")
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
