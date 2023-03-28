package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
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

func RunCmd(bin string, args []string) (string, error) {
	cmd := exec.Command(bin, args...)
	cmdOut, err := cmd.CombinedOutput()
	if err != nil {
		var exitError *exec.ExitError
		if !errors.As(err, &exitError) {
			return "", fmt.Errorf("failed to run %s: %w", filepath.Base(bin), err)
		}
	}
	if cmd.ProcessState == nil {
		panic("cmd.ProcessState should not be nil after running")
	}
	exitCode := cmd.ProcessState.ExitCode()
	cmdOutStr := string(cmdOut)
	cmdOutStr = strings.TrimSpace(cmdOutStr)
	if exitCode != 0 {
		return cmdOutStr, fmt.Errorf("%s error: %s", filepath.Base(bin), cmdOutStr)
	}
	return cmdOutStr, nil
}
