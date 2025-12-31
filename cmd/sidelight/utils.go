package main

import (
	"os"
	"path/filepath"
	"strings"
	"log"
)

func isSupportedFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	// Common RAW extensions + Standard Images
	exts := map[string]bool{
		// RAW
		".arw": true,
		".cr2": true,
		".cr3": true,
		".nef": true,
		".orf": true,
		".dng": true,
		".raf": true,
		// Standard
		".jpg":  true,
		".jpeg": true,
		".png":  true,
	}
	return exts[ext]
}

func collectFiles(args []string) []string {
	var files []string
	for _, arg := range args {
		info, err := os.Stat(arg)
		if err != nil {
			log.Printf("Warning: could not access %s: %v", arg, err)
			continue
		}
		if info.IsDir() {
			err := filepath.Walk(arg, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}
				if !info.IsDir() && isSupportedFile(path) {
					files = append(files, path)
				}
				return nil
			})
			if err != nil {
				log.Printf("Warning: error walking directory %s: %v", arg, err)
			}
		} else {
			if isSupportedFile(arg) {
				files = append(files, arg)
			}
		}
	}
	return files
}
