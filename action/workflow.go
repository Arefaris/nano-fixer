package action

import (
	"context"
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/gen2brain/beeep"
	"personal-grammarly/ai"
	"personal-grammarly/keyboard"
)

func RunCorrection(aiClient *ai.Client) {
	log.Println("Correction triggered")

	// 1. Save current clipboard
	originalClipboard, err := clipboard.ReadAll()
	if err != nil {
		log.Println("Warning: Could not read original clipboard:", err)
		originalClipboard = ""
	}

	// 2. Clear clipboard so we know when new text is ready
	clipboard.WriteAll("")

	// 3. Simulate Ctrl+C
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

	log.Println("Selected text:", selectedText)
	notify("Processing", "Correcting grammar...")

	// 6. Call AI API
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	correctedText, err := aiClient.CorrectText(ctx, selectedText)
	if err != nil {
		log.Println("AI API Error:", err)
		notify("Error", "Failed to correct text.")
		restoreClipboard(originalClipboard)
		return
	}

	log.Println("Corrected text:", correctedText)

	// 7. Write corrected text to clipboard
	err = clipboard.WriteAll(correctedText)
	if err != nil {
		log.Println("Error writing to clipboard:", err)
		restoreClipboard(originalClipboard)
		return
	}

	// 8. Simulate Ctrl+V to replace text
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
		// Just clear it
		clipboard.WriteAll("")
	}
}

func notify(title, message string) {
	err := beeep.Notify(title, message, "")
	if err != nil {
		log.Println("Notification error:", err)
	}
}
