package main

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	const templateName = "template.dmg"

	// create dmg from template
	const tmpDir = "./tmp"

	err := os.Mkdir(tmpDir, 0755)
	if err != nil {
		log.Fatalln(err)
	}

	defer os.RemoveAll(tmpDir)

	const tmpDMG = "./tmp.dmg"

	// copy the template image, since we'll be modifying it
	err = copyFile(templateName, tmpDMG, nil)
	if err != nil {
		log.Fatalf("making copy of template DMG: %v", err)
	}
	defer os.Remove(tmpDMG)

	// attach the template dmg
	cmd := exec.Command("hdiutil", "attach", tmpDMG, "-noautoopen", "-mountpoint", tmpDir)
	attachBuf := new(bytes.Buffer)
	cmd.Stdout = attachBuf
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("running hdiutil attach: %v", err)
	}

	version := "dev"
	if ref, ok := os.LookupEnv("GITHUB_REF_NAME"); ok {
		version = ref
	}
	err = os.WriteFile("Couper.app/Contents/version.plist", []byte(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
    <dict>
        <key>CFBundleShortVersionString</key>
        <string>`+version+`</string>
        <key>ProjectName</key>
        <string>Couper</string>
    </dict>
</plist>
`), 0755)
	if err != nil {
		log.Fatalf("error writing version property file: %v", err)
	}

	// move bundle file into it
	err = deepCopy("Couper.app", tmpDir)
	if err != nil {
		log.Fatalf("copying app into dmg: %v", err)
	}

	// get attached image's device; it should be the
	// first device that is outputted
	hdiutilOutFields := strings.Fields(attachBuf.String())
	if len(hdiutilOutFields) == 0 {
		log.Fatalf("no device output by hdiutil attach")
	}
	dmgDevice := hdiutilOutFields[0]

	// detach image
	cmd = exec.Command("hdiutil", "detach", dmgDevice)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("running hdiutil detach: %v", err)
	}

	// convert to compressed image
	outputDMG := "couper.dmg"
	cmd = exec.Command("hdiutil", "convert", tmpDMG, "-format", "UDZO", "-imagekey", "zlib-level=9", "-o", outputDMG)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		log.Fatalf("running hdiutil convert: %v", err)
	}
}

func copyFile(from, to string, fromInfo os.FileInfo) error {
	log.Printf("[INFO] Copying %s to %s", from, to)

	if fromInfo == nil {
		var err error
		fromInfo, err = os.Stat(from)
		if err != nil {
			return err
		}
	}

	// open source file
	fsrc, err := os.Open(from)
	if err != nil {
		return err
	}

	// create destination file, with identical permissions
	fdest, err := os.OpenFile(to, os.O_RDWR|os.O_CREATE|os.O_TRUNC, fromInfo.Mode()&os.ModePerm)
	if err != nil {
		fsrc.Close()
		if _, err2 := os.Stat(to); err2 == nil {
			return fmt.Errorf("opening destination (which already exists): %v", err)
		}
		return err
	}

	// copy the file and ensure it gets flushed to disk
	if _, err = io.Copy(fdest, fsrc); err != nil {
		fsrc.Close()
		fdest.Close()
		return err
	}
	if err = fdest.Sync(); err != nil {
		fsrc.Close()
		fdest.Close()
		return err
	}

	// close both files
	if err = fsrc.Close(); err != nil {
		fdest.Close()
		return err
	}
	if err = fdest.Close(); err != nil {
		return err
	}

	return nil
}

// deepCopy makes a deep copy of from into to.
func deepCopy(from, to string) error {
	if from == "" || to == "" {
		return fmt.Errorf("no source or no destination; both required")
	}

	// traverse the source directory and copy each file
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		// error accessing current file
		if err != nil {
			return err
		}

		// skip files/folders without a name
		if info.Name() == "" {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// if directory, create destination directory (if not
		// already created by our pre-walk)
		if info.IsDir() {
			subdir := strings.TrimPrefix(path, filepath.Dir(from))
			destDir := filepath.Join(to, subdir)
			if _, err := os.Stat(destDir); os.IsNotExist(err) {
				err := os.Mkdir(destDir, info.Mode()&os.ModePerm)
				if err != nil {
					return err
				}
			}
			return nil
		}

		destPath := filepath.Join(to, strings.TrimPrefix(path, filepath.Dir(from)))
		err = copyFile(path, destPath, info)
		if err != nil {
			return fmt.Errorf("copying file %s: %v", path, err)
		}
		return nil
	})
}
