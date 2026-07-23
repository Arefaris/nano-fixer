package action

import (
	"context"
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"nano-fixer/ai"
	"nano-fixer/config"
	"nano-fixer/gui"
	"nano-fixer/keyboard"
	"nano-fixer/localai"
)

func RunCorrection(aiClient *ai.Client, getConfig func() *config.Config) {
	cfg := getConfig()
	log.Println("Correction triggered. Language:", cfg.TargetLanguage)

	// 1. Save current clipboard
	originalClipboard, err := clipboard.ReadAll()
	if err != nil {
		log.Println("Warning: Could not read original clipboard:", err)
		originalClipboard = ""
	}

	// 2. Clear clipboard so we know when new text is ready
	clipboard.WriteAll("")

	// 3. Simulate Ctrl+C (with modifier release)
	err = keyboard.SimulateCopy()
	if err != nil {
		log.Println("Error simulating copy:", err)
		notify("Error", "Could not copy text.")
		return
	}

	// 4. Wait a bit for clipboard to populate (retry loop)
	var selectedText string
	for i := 0; i < 15; i++ {
		time.Sleep(50 * time.Millisecond)
		selectedText, err = clipboard.ReadAll()
		if err == nil && selectedText != "" {
			break
		}
	}

	// 5. Check if we got text
	if selectedText == "" {
		log.Println("No text selected or could not read clipboard")
		restoreClipboard(originalClipboard)
		return
	}

	log.Printf("Selected text length: %d characters\n", len(selectedText))
	// notify("Processing", "Correcting grammar...")
	gui.ShowHUD("✨ AI is fixing...")
	defer gui.HideHUD()

	// 6. Ensure Local AI is running if enabled
	if cfg.UseLocalAI && !localai.IsRunning() {
		log.Println("Local AI engine is not running (crashed or sleeping). Restarting it...")
		gui.ShowHUD("⚙️ Restarting AI Engine...")
		err := localai.StartEngine()
		if err != nil {
			log.Println("Failed to restart engine:", err)
			notify("Error", "Could not restart local AI engine.")
			restoreClipboard(originalClipboard)
			return
		}
		gui.ShowHUD("✨ AI is fixing...")
	}

	// 7. Call AI API
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	correctedText, err := aiClient.CorrectText(ctx, selectedText, cfg.TargetLanguage)
	if err != nil {
		log.Println("AI API Error:", err)
		notify("Error", "Failed to correct text.")
		restoreClipboard(originalClipboard)
		return
	}

	log.Printf("Corrected text length: %d characters\n", len(correctedText))

	// 7. Write corrected text to clipboard
	err = clipboard.WriteAll(correctedText)
	if err != nil {
		log.Println("Error writing to clipboard:", err)
		restoreClipboard(originalClipboard)
		return
	}

	// 8. Simulate Ctrl+V to replace text (with modifier release)
	err = keyboard.SimulatePaste()
	if err != nil {
		log.Println("Error simulating paste:", err)
	} else {
		notify("Success", "Text corrected!")
	}

	// 9. Restore original clipboard
	// Need to wait slightly for the OS to consume the paste before restoring
	time.Sleep(200 * time.Millisecond)
	restoreClipboard(originalClipboard)
}

func restoreClipboard(text string) {
	if text != "" {
		err := clipboard.WriteAll(text)
		if err != nil {
			log.Println("Warning: failed to restore clipboard:", err)
		}
	} else {
		clipboard.WriteAll("")
	}
}

func notify(title, message string) {
	// Using the app icon in notifications is supported by beeep on some platforms,
	// but we can just use the standard toast notification.
	err := beeep.Notify(title, message, "")
	if err != nil {
		log.Println("Notification error:", err)
	}
}
