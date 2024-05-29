package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ExiftoolProcess returns list of files successfully processed, and map of filename -> error.
func ExiftoolProcess(ctx context.Context, args []string, files []string, appConfig AppConfig, verbose bool, verbose2 bool) ([]string, map[string]error) {
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

		cmdOut, err := RunCmd(appConfig.ExiftoolBin, fullArgs)
		if err != nil {
			errs[imgFilename] = err
			ErrPrint(ctx, errs[imgFilename])
			continue
		}
		if verbose {
			fmt.Println(cmdOut)
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
				ErrPrintln(ctx, "could not stat exiftool backup file '%s': %s\n", exiftoolBackupFilename, err)
			}
		} else {
			backupsConfig, err := GetBackupConfig(imgFilename)
			if err != nil {
				errs[imgFilename] = fmt.Errorf("failed to get backups config: %w", err)
				ErrPrint(ctx, errs[imgFilename])
				continue
			}
			backupsPath, err := backupsConfig.PrepareBackupsDir(imgFilename, startTime)
			if err != nil {
				errs[imgFilename] = fmt.Errorf("failed to prepare backups folder: %w", err)
				ErrPrint(ctx, errs[imgFilename])
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
					ErrPrint(ctx, errs[imgFilename])
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
