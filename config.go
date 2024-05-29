package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//goland:noinspection GoDeprecation
type AppConfig struct {
	ExiftoolBin    string            `json:"exiftool_bin,omitempty"` // absolute path to exiftool
	CamswapAliases map[string]string `json:"camswap_aliases,omitempty"`
	NeatImage      struct {
		NeatImageBin      string `json:"neat_image_bin,omitempty"`
		ProfilesFolder    string `json:"profiles_folder"`
		DefaultJpgQuality int    `json:"default_jpg_quality"`
	} `json:"neat_image,omitempty"`
	DeprecatedX3fBin string `json:"x3f_bin,omitempty"` // deprecated; retained here for backward compatibility
	X3fExtractBin    string `json:"x3f_extract_bin,omitempty"`
}

func buildAppConfig(ctx context.Context) (AppConfig, error) {
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
				return appConfig, fmt.Errorf("failed to read xtoolconfig file '%s': '%w'", configPath, err)
			}
		}

		// Parse the discovered xtoolconfig file:
		err = json.Unmarshal(appConfigBytes, &appConfig)
		if err != nil {
			return appConfig, fmt.Errorf("failed to parse '%s' as JSON: %w", configPath, err)
		}
		break
	}

	// Fallback to finding exiftool in the path if it wasn't specified in a config:

	if appConfig.ExiftoolBin == "" {
		exiftoolPath, err := exec.LookPath("exiftool")
		if err != nil {
			ErrPrintln(ctx, "exiftool_bin was not specified in config and is missing from $PATH")
			return appConfig, fmt.Errorf("$PATH search failed with: %w", err)
		}
		appConfig.ExiftoolBin = exiftoolPath
	}

	// Validate the xtoolconfig:

	if stat, err := os.Stat(appConfig.ExiftoolBin); err != nil {
		return appConfig, fmt.Errorf("bad path to exiftool binary '%s': %w", appConfig.ExiftoolBin, err)
	} else if !IsExecAny(stat.Mode()) {
		return appConfig, fmt.Errorf("exiftool at '%s' is not executable", appConfig.ExiftoolBin)
	}

	if appConfig.CamswapAliases == nil {
		appConfig.CamswapAliases = make(map[string]string)
	}

	// config is valid!
	return appConfig, nil
}

func (c AppConfig) GetX3fExtractBin() string {
	if c.X3fExtractBin != "" {
		return c.X3fExtractBin
	}
	return c.DeprecatedX3fBin
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

var backupConfigCache = make(map[string]BackupsConfig)

func GetBackupConfig(filename string) (BackupsConfig, error) {
	// Finding the applicable .xtoolbak config file, we search upward starting at the directory the image file is in:
	// - If under `~`: search stops at ~.
	// - If under /Volumes, /mnt, /media: search stops at the volume root.
	// - Else: search stops at root.
	// If no backup config file is found, a default configuration is returned.

	homeDir := MustUserHomeDir()
	absImageFilePath, err := filepath.Abs(filename)
	if err != nil {
		return BackupsConfig{}, fmt.Errorf("failed to find absolute path for '%s': %w", filename, err)
	}
	bakConfigSearchDir := filepath.Dir(absImageFilePath)
	bakConfigSearchVolName := filepath.VolumeName(bakConfigSearchDir)

	if cachedConfig, ok := backupConfigCache[bakConfigSearchDir]; ok {
		return cachedConfig, nil
	}

	i := 0
	for {
		if i > 128 {
			return BackupsConfig{}, fmt.Errorf("failed to find a backups config for '%s' in 128 iterations", filename)
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
			return backupsConfig, fmt.Errorf("failed to read '%s': %w", backupsConfigPath, err)
		}
	} else {
		err = json.Unmarshal(backupsConfigBytes, &backupsConfig)
		if err != nil {
			return backupsConfig, fmt.Errorf("failed to parse '%s' as JSON: %w", backupsConfigPath, err)
		}
	}

	// Validate backups config:

	if backupsConfig.BackupsLocation != BackupsLocSameDir && backupsConfig.BackupsLocation != BackupsLocSubDir && backupsConfig.BackupsLocation != BackupsLocAbsPath {
		return backupsConfig, fmt.Errorf("backups_location must be one of (same_dir, sub_dir, abs_path); got '%s'", backupsConfig.BackupsLocation)
	}

	if backupsConfig.BackupsLocation == BackupsLocSubDir && backupsConfig.BackupsFolder == "" {
		return backupsConfig, errors.New("'backups_location: sub_dir' requires setting a backups_folder, to name the backups subdirectory")
	}

	if backupsConfig.BackupsLocation == BackupsLocSubDir && strings.Contains(backupsConfig.BackupsFolder, string(os.PathSeparator)) {
		return backupsConfig, errors.New("backups_folder must be a simple directory name for 'backups_location: sub_dir'")
	}

	if backupsConfig.BackupsLocation == BackupsLocAbsPath {
		if backupsConfig.BackupsFolder == "" {
			return backupsConfig, errors.New("'backups_location: abs_path' requires setting backups_folder to an absolute path")
		}
		if stat, err := os.Stat(backupsConfig.BackupsFolder); err != nil {
			return backupsConfig, fmt.Errorf("bad backups_folder '%s': %s", backupsConfig.BackupsFolder, err)
		} else if !stat.IsDir() {
			return backupsConfig, fmt.Errorf("bad backups_folder '%s': is not a directory", backupsConfig.BackupsFolder)
		}
	}

	backupConfigCache[filepath.Dir(absImageFilePath)] = backupsConfig
	return backupsConfig, nil
}
