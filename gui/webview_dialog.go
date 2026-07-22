package gui

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	webview2 "github.com/jchv/go-webview2"
	"nano-fixer/config"
	"nano-fixer/localai"
)

var (
	wvMutex  sync.Mutex
	activeWv webview2.WebView
)

func ShowWebViewSettings(uiFS embed.FS, cfgGetter func() *config.Config, onSave func(newCfg config.Config)) {
	wvMutex.Lock()
	if activeWv != nil {
		wvMutex.Unlock()
		return
	}
	wvMutex.Unlock()

	go func() {
		runtime.LockOSThread()
		
		w := webview2.New(false)
		if w == nil {
			log.Println("Error: Failed to initialize WebView2 window.")
			return
		}

		wvMutex.Lock()
		activeWv = w
		wvMutex.Unlock()

		defer func() {
			wvMutex.Lock()
			activeWv = nil
			wvMutex.Unlock()
			w.Destroy()
		}()

		w.SetTitle("Nano Fixer - Settings")
		w.SetSize(460, 680, webview2.HintFixed)

		w.Bind("getConfig", func() string {
			cfg := cfgGetter()
			cfgCopy := *cfg
			if cfgCopy.APIKey != "" {
				cfgCopy.APIKey = "••••••••••••"
			}
			data, _ := json.Marshal(cfgCopy)
			return string(data)
		})

		w.Bind("saveConfig", func(jsonStr string) bool {
			var newCfg config.Config
			err := json.Unmarshal([]byte(jsonStr), &newCfg)
			if err != nil {
				log.Println("Failed to parse config JSON from webview:", err)
				return false
			}
			currentCfg := cfgGetter()
			if newCfg.APIKey == "••••••••••••" || newCfg.APIKey == "" {
				newCfg.APIKey = currentCfg.APIKey
			}
			onSave(newCfg)
			return true
		})

		w.Bind("openLogs", func() {
			exePath, err := os.Executable()
			if err == nil {
				logFilePath := filepath.Join(filepath.Dir(exePath), "app.log")
				go exec.Command("notepad.exe", logFilePath).Start()
			}
		})

		w.Bind("checkLocalAIFiles", func() bool {
			return localai.CheckFilesExist()
		})

		w.Bind("startLocalAIDownload", func() {
			go func() {
				err := localai.EnsureLocalAIFiles()
				if err != nil {
					log.Println("Local AI download failed:", err)
				}
			}()
		})

		w.Bind("getLocalAIDownloadProgress", func() string {
			pct, status, downloading := localai.GetProgress()
			res := map[string]interface{}{
				"pct":         pct,
				"status":      status,
				"downloading": downloading,
			}
			data, _ := json.Marshal(res)
			return string(data)
		})

		w.Bind("closeWindow", func() {
			w.Destroy()
		})

		// Read files from embed FS
		htmlBytes, _ := uiFS.ReadFile("settings_ui/index.html")
		cssBytes, _ := uiFS.ReadFile("settings_ui/style.css")
		jsBytes, _ := uiFS.ReadFile("settings_ui/app.js")

		htmlContent := string(htmlBytes)
		// Inline CSS and JS into single standalone HTML document for Webview
		htmlContent = strings.Replace(htmlContent, `<link rel="stylesheet" href="style.css">`, `<style>`+string(cssBytes)+`</style>`, 1)
		htmlContent = strings.Replace(htmlContent, `<script src="app.js"></script>`, `<script>`+string(jsBytes)+`</script>`, 1)

		encodedHTML := base64.StdEncoding.EncodeToString([]byte(htmlContent))
		w.Navigate("data:text/html;base64," + encodedHTML)
		w.Run()
	}()
}
