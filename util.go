package main

import (
	"fmt"
	"os"
	"path/filepath"
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
