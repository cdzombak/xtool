package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type AppConfig struct {
	ExiftoolBin    string            `json:"exiftool_bin,omitempty"` // absolute path to exiftool
	CamswapAliases map[string]string `json:"camswap_aliases,omitempty"`
}

type BackupsConfig struct {
	BackupsLocation string `json:"backups_location"`         // same_dir, sub_dir, abs_path. same_dir = exiftool default; sub_dir = move exiftool backup files to a subdirectory; abs_path = move backups to a structure under an absolute path
	BackupsFolder   string `json:"backups_folder,omitempty"` // same_dir = no effect; sub_dir = backups at ./backups_folder_TS; abs_path = backups at abs_path/TS source_folder_name
}

const (
	backupsConfigName = ".xtoolbak.json"

	BackupsLocSameDir = "same_dir"
	BackupsLocSubDir  = "sub_dir"
	BackupsLocAbsPath = "abs_path"
)

func GetAppConfig() AppConfig {
	homeDir := MustUserHomeDir()

	// Finding the applicable .xtoolconfig file: the following paths are checked, in this order:
	// - ~/.config/xtoolconfig
	// - ~/.xtoolconfig

	appConfig := AppConfig{}
	appConfigPaths := []string{
		filepath.Join(homeDir, ".config", "xtoolconfig.json"),
		filepath.Join(homeDir, ".xtoolconfig.json"),
	}
	for _, configPath := range appConfigPaths {
		appConfigBytes, err := os.ReadFile(configPath)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			} else {
				fmt.Printf("failed to read xtoolconfig file '%s': '%s'\n", configPath, err)
				os.Exit(1)
			}
		}

		// Parse the discovered xtoolconfig file:
		err = json.Unmarshal(appConfigBytes, &appConfig)
		if err != nil {
			fmt.Printf("failed to parse '%s' as JSON: %s\n", configPath, err)
			os.Exit(1)
		}
		break
	}

	// Fallback to finding exiftool in the path if it wasn't specified in a config:

	if appConfig.ExiftoolBin == "" {
		exiftoolPath, err := exec.LookPath("exiftool")
		if err != nil {
			fmt.Println("exiftool_bin was not specified in config and is missing from $PATH")
			fmt.Printf("$PATH search failed with: %s\n", err)
			os.Exit(1)
		}
		appConfig.ExiftoolBin = exiftoolPath
	}

	// Validate the xtoolconfig:

	if stat, err := os.Stat(appConfig.ExiftoolBin); err != nil {
		fmt.Printf("bad path to exiftool binary '%s': %s\n", appConfig.ExiftoolBin, err)
		os.Exit(1)
	} else if !IsExecAny(stat.Mode()) {
		fmt.Printf("exiftool at '%s' is not executable\n", appConfig.ExiftoolBin)
		os.Exit(1)
	}

	if appConfig.CamswapAliases == nil {
		appConfig.CamswapAliases = make(map[string]string)
	}

	// config is valid!
	return appConfig
}

var backupConfigCache = make(map[string]BackupsConfig)

func GetBackupConfig(filename string) BackupsConfig {
	// Finding the applicable .xtoolbak config file, we search upward starting at the directory the image file is in:
	// - If under `~`: search stops at ~.
	// - If under /Volumes, /mnt, /media: search stops at the volume root.
	// - Else: search stops at root.
	// If no backup config file is found, a default configuration is returned.

	homeDir := MustUserHomeDir()
	absImageFilePath, err := filepath.Abs(filename)
	if err != nil {
		fmt.Printf("failed to find absolute path for '%s': %s\n", filename, err)
		os.Exit(1)
	}
	bakConfigSearchDir := filepath.Dir(absImageFilePath)
	bakConfigSearchVolName := filepath.VolumeName(bakConfigSearchDir)

	if cachedConfig, ok := backupConfigCache[bakConfigSearchDir]; ok {
		return cachedConfig
	}

	i := 0
	for {
		if i > 128 {
			fmt.Printf("failed to find a backups config for '%s' in 128 iterations\n", filename)
			os.Exit(1)
		}

		if bakConfigSearchDir == homeDir || bakConfigSearchDir == bakConfigSearchVolName ||
			bakConfigSearchDir == "/" || filepath.Dir(bakConfigSearchDir) == "/Volumes" ||
			filepath.Dir(bakConfigSearchDir) == "/mnt" || filepath.Dir(bakConfigSearchDir) == "/media" ||
			filepath.Dir(bakConfigSearchDir) == "/usb" {
			break
		}

		configCandidatePath := filepath.Join(bakConfigSearchDir, backupsConfigName)
		_, err := os.Stat(configCandidatePath)
		if err == nil {
			break
		}

		bakConfigSearchDir = filepath.Dir(bakConfigSearchDir)
		i++
	}

	// Read and parse the relevant .xtoolbak file (or set a default config if none is found):

	backupsConfig := BackupsConfig{}
	backupsConfigPath := filepath.Join(bakConfigSearchDir, backupsConfigName)
	backupsConfigBytes, err := os.ReadFile(backupsConfigPath)
	if err != nil {
		if os.IsNotExist(err) {
			backupsConfig.BackupsLocation = BackupsLocSameDir
		} else {
			fmt.Printf("Failed to read '%s': %s\n", backupsConfigPath, err)
			os.Exit(1)
		}
	} else {
		err = json.Unmarshal(backupsConfigBytes, &backupsConfig)
		if err != nil {
			fmt.Printf("failed to parse '%s' as JSON: %s\n", backupsConfigPath, err)
			os.Exit(1)
		}
	}

	// Validate backups config:

	if backupsConfig.BackupsLocation != BackupsLocSameDir && backupsConfig.BackupsLocation != BackupsLocSubDir && backupsConfig.BackupsLocation != BackupsLocAbsPath {
		fmt.Printf("backups_location must be one of (same_dir, sub_dir, abs_path); got '%s'\n", backupsConfig.BackupsLocation)
		os.Exit(1)
	}

	if backupsConfig.BackupsLocation == BackupsLocSubDir && backupsConfig.BackupsFolder == "" {
		fmt.Println("'backups_location: sub_dir' requires setting a backups_folder, to name the backups subdirectory")
		os.Exit(1)
	}

	if backupsConfig.BackupsLocation == BackupsLocSubDir && strings.Contains(backupsConfig.BackupsFolder, string(os.PathSeparator)) {
		fmt.Println("backups_folder must be a simple directory name for 'backups_location: sub_dir'")
		os.Exit(1)
	}

	if backupsConfig.BackupsLocation == BackupsLocAbsPath {
		if backupsConfig.BackupsFolder == "" {
			fmt.Println("'backups_location: abs_path' requires setting backups_folder to an absolute path")
			os.Exit(1)
		}
		if stat, err := os.Stat(backupsConfig.BackupsFolder); err != nil {
			fmt.Printf("bad backups_folder '%s': %s", backupsConfig.BackupsFolder, err)
			os.Exit(1)
		} else if !stat.IsDir() {
			fmt.Printf("bad backups_folder '%s': is not a directory", backupsConfig.BackupsFolder)
			os.Exit(1)
		}
	}

	backupConfigCache[filepath.Dir(absImageFilePath)] = backupsConfig
	return backupsConfig
}
