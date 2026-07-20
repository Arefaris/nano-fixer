package main

import (
	"log"
	"os"
	"os/exec"
)

const manifestContent = `<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<assembly xmlns="urn:schemas-microsoft-microsoft-com:asm.v1" manifestVersion="1.0">
<assemblyIdentity
    version="1.0.0.0"
    processorArchitecture="*"
    name="NanoFixer"
    type="win32"
/>
<description>Nano Fixer - Background Grammar Corrector</description>
<dependency>
    <dependentAssembly>
        <assemblyIdentity
            type="win32"
            name="Microsoft.Windows.Common-Controls"
            version="6.0.0.0"
            processorArchitecture="*"
            publicKeyToken="6595b64144ccf1df"
            language="*"
        />
    </dependentAssembly>
</dependency>
</assembly>
`

func main() {
	log.Println("Checking for rsrc tool...")

	// Install rsrc tool
	cmd := exec.Command("go", "install", "github.com/akavel/rsrc@latest")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		log.Fatalf("Failed to install rsrc: %v", err)
	}

	log.Println("Creating app.manifest...")
	err = os.WriteFile("app.manifest", []byte(manifestContent), 0644)
	if err != nil {
		log.Fatalf("Failed to create app.manifest: %v", err)
	}

	log.Println("Compiling resources into rsrc.syso...")
	
	// Get go path bin directory
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		homeDir, err := os.UserHomeDir()
		if err == nil {
			goPath = os.Getenv("USERPROFILE")
			if goPath == "" {
				goPath = homeDir
			}
			goPath = goPath + "/go"
		}
	}
	rsrcPath := "rsrc"
	if goPath != "" {
		rsrcPath = goPath + "/bin/rsrc"
	}

	cmd = exec.Command(rsrcPath, "-ico", "icon.ico", "-o", "rsrc.syso")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		// Fallback to searching rsrc in path
		cmd = exec.Command("rsrc", "-ico", "icon.ico", "-o", "rsrc.syso")
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err = cmd.Run()
		if err != nil {
			log.Fatalf("Failed to run rsrc: %v. Make sure %%GOPATH%%\\bin is in your PATH.", err)
		}
	}

	log.Println("rsrc.syso generated successfully!")
}
