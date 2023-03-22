package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
)

func (c BackupsConfig) PrepareBackupsDir(filename string, startTime time.Time) (string, error) {
	backupsPath := ""
	ts := startTime.Format("2006-01-02T15-04-05")
	absFilePath, err := filepath.Abs(filename)
	if err != nil {
		return "", err
	}
	var parentMode os.FileMode
	if c.BackupsLocation == BackupsLocSubDir {
		backupsPath = filepath.Join(
			filepath.Dir(absFilePath),
			fmt.Sprintf("%s_%s", c.BackupsFolder, ts),
		)
		stat, err := os.Stat(filepath.Dir(absFilePath))
		if err != nil {
			return "", err
		}
		parentMode = stat.Mode() & os.ModePerm
	} else if c.BackupsLocation == BackupsLocAbsPath {
		backupsPath = filepath.Join(
			c.BackupsFolder,
			fmt.Sprintf("%s %s", ts, filepath.Base(filepath.Dir(absFilePath))),
		)
		stat, err := os.Stat(c.BackupsFolder)
		if err != nil {
			return "", err
		}
		parentMode = stat.Mode() & os.ModePerm
	}
	if backupsPath != "" {
		err := os.MkdirAll(backupsPath, parentMode)
		if err != nil {
			fmt.Printf("failed to create backups directory '%s': %s\n", backupsPath, err)
			os.Exit(1)
		}
	}
	return backupsPath, nil
}

// ExiftoolProcess returns list of files successfully processed, and map of filename -> error.
func ExiftoolProcess(args []string, files []string, appConfig AppConfig, verbose bool, verbose2 bool) ([]string, map[string]error) {
	errorPrintln := color.New(color.FgRed).PrintlnFunc()

	var successes []string
	errs := make(map[string]error)
	startTime := time.Now()

	for _, imgFilename := range files {
		fmt.Printf("%s ...\n", imgFilename)

		fullArgs := make([]string, len(args)+1)
		copy(fullArgs, args)
		fullArgs[len(args)] = imgFilename

		if verbose2 {
			fmt.Printf("%s %s\n", appConfig.ExiftoolBin, strings.Join(fullArgs, " "))
		}

		cmd := exec.Command(appConfig.ExiftoolBin, fullArgs...)
		cmdOut, err := cmd.CombinedOutput()
		if err != nil {
			var exitError *exec.ExitError
			if !errors.As(err, &exitError) {
				errs[imgFilename] = fmt.Errorf("failed to run exiftool: %w", err)
				errorPrintln(errs[imgFilename].Error())
				continue
			}
		}
		if cmd.ProcessState == nil {
			panic("cmd.ProcessState should not be nil after running")
		}
		exitCode := cmd.ProcessState.ExitCode()
		cmdOutStr := string(cmdOut)
		cmdOutStr = strings.TrimSpace(cmdOutStr)
		if exitCode != 0 {
			errs[imgFilename] = fmt.Errorf("exiftool error: %s", cmdOutStr)
			errorPrintln(errs[imgFilename].Error())
			continue
		}
		if verbose {
			fmt.Println(cmdOutStr)
		}

		exiftoolBackupFilename := fmt.Sprintf("%s_original", imgFilename)
		_, err = os.Stat(exiftoolBackupFilename)
		if err != nil {
			if os.IsNotExist(err) {
				// backup file was not created; move on. (supports -s)
				if verbose2 {
					fmt.Printf("exiftool backup file '%s' does not exist; nothing to do\n", exiftoolBackupFilename)
				}
			} else {
				fmt.Printf("could not stat exiftool backup file '%s': %s\n", exiftoolBackupFilename, err)
			}
		} else {
			backupsConfig := GetBackupConfig(imgFilename)
			backupsPath, err := backupsConfig.PrepareBackupsDir(imgFilename, startTime)
			if err != nil {
				errs[imgFilename] = fmt.Errorf("failed to prepare backups folder: %w", err)
				errorPrintln(errs[imgFilename].Error())
				continue
			}
			if backupsPath != "" {
				newBackupFilePath := filepath.Join(backupsPath, filepath.Base(imgFilename))
				err = os.Rename(
					exiftoolBackupFilename,
					newBackupFilePath,
				)
				if err != nil {
					errs[imgFilename] = fmt.Errorf("failed to move backup file '%s' to the backups folder: %w", exiftoolBackupFilename, err)
					errorPrintln(errs[imgFilename].Error())
					continue
				}
				if verbose2 {
					fmt.Printf("Moved exiftool backup file '%s' to '%s'.", exiftoolBackupFilename, newBackupFilePath)
				}
			}
		}

		successes = append(successes, imgFilename)
	}

	return successes, errs
}

func MustUserHomeDir() string {
	retv, err := os.UserHomeDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	return filepath.Clean(retv)
}

func IsExecAny(mode os.FileMode) bool {
	return mode&0111 != 0
}
