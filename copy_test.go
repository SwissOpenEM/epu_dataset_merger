package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestMyFileOperations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)
	srcDir, err := filepath.Abs("testing")
	if err != nil {
		log.Fatalf("That went wrong, %s", err)
	}
	dstDir := filepath.Join(tempDir, "sample_folder_copy")

	if err := copyDir(srcDir, dstDir); err != nil {
		t.Fatalf("failed to copy test folder: %v", err)
	}

	atlasdir := dstDir
	datadir := filepath.Join(dstDir, "X")
	epudir := filepath.Join(dstDir, "Y")
	if err := syncXMLFromYtoX(datadir, epudir, 8, atlasdir); err != nil {
		t.Errorf("runFileOperations failed: %v", err)
	}
	expectedFolder, err := filepath.Abs("CorrectTarget")
	if err != nil {
		log.Fatalf("Correct data directory was not located, %s", err)
	}
	//Compare if result matches correct output
	if err := compareDirectories(expectedFolder, datadir); err != nil {
		t.Errorf("Directory comparison failed: %v", err)
	}
}

func compareDirectories(expectedDir, targetDir string) error {
	err := filepath.Walk(expectedDir, func(expPath string, expInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(expectedDir, expPath)
		if err != nil {
			return err
		}
		targetPath := filepath.Join(targetDir, relPath)
		targetInfo, err := os.Stat(targetPath)
		if err != nil {
			return fmt.Errorf("expected file %s not found in target (%v)", relPath, err)
		}

		if expInfo.IsDir() != targetInfo.IsDir() {
			return fmt.Errorf("mismatch for %s: one is a directory, the other is not", relPath)
		}
		if !expInfo.IsDir() {
			expData, err := ioutil.ReadFile(expPath)
			if err != nil {
				return fmt.Errorf("error reading expected file %s: %v", relPath, err)
			}
			targetData, err := ioutil.ReadFile(targetPath)
			if err != nil {
				return fmt.Errorf("error reading target file %s: %v", relPath, err)
			}
			if !bytes.Equal(expData, targetData) {
				return fmt.Errorf("file %s differs between expected and target", relPath)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	//check for extra files in targetDir.
	err = filepath.Walk(targetDir, func(targetPath string, targetInfo os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(targetDir, targetPath)
		if err != nil {
			return err
		}
		expectedPath := filepath.Join(expectedDir, relPath)
		if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
			return fmt.Errorf("extra file or directory %s found in target", relPath)
		}
		return nil
	})
	return err
}
