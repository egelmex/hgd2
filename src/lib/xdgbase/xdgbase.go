package xdgbase

import (
	"os"
	"strings"
)

var XDG_CONFIG_HOME string
var XDG_DATA_HOME string
var XDG_DATA_DIRS []string
var XDG_CONFIG_DIRS []string
var XDG_CACHE_HOME string

func init() {
}

func GetConfigHome() string {
	if XDG_CONFIG_HOME == "" {
		XDG_CONFIG_HOME = os.Getenv("XDG_CONFIG_HOME")
		if XDG_CONFIG_HOME == "" {
			XDG_CONFIG_HOME = os.Getenv("HOME") + "/.config"
		}
	}
	return XDG_CONFIG_HOME
}

func GetCacheHome() string {
	if XDG_CACHE_HOME == "" {
		XDG_CACHE_HOME = os.Getenv("XDG_CACHE_HOME")
		if XDG_CACHE_HOME == "" {
			XDG_CACHE_HOME = os.Getenv("HOME") + "/.cache"
		}
	}
	return XDG_CONFIG_HOME
}

func GetDataHome() string {
	if XDG_DATA_HOME == "" {
		XDG_DATA_HOME = os.Getenv("XDG_DATA_HOME")
		if XDG_DATA_HOME == "" {
			XDG_DATA_HOME = os.Getenv("HOME") + "/.local/share"
		}
	}
	return XDG_DATA_HOME
}

func GetDataDirs() []string {
	if XDG_DATA_DIRS == nil {
		dirs := os.Getenv("XDG_DATA_DIRS")
		if dirs != "" {
			XDG_DATA_DIRS = strings.Split(dirs, ":")
		} else {
			XDG_DATA_DIRS = []string{"/usr/local/share/", "/usr/share/"}
		}
	}
	return XDG_DATA_DIRS
}

func GetConfigDirs() []string {
	if XDG_CONFIG_DIRS == nil {
		dirs := os.Getenv("XDG_CONFIG_DIRS")
		if dirs != "" {
			XDG_CONFIG_DIRS = strings.Split(dirs, ":")
		} else {
			XDG_CONFIG_DIRS = []string{"/etc/xdg"}
		}
	}
	return XDG_CONFIG_DIRS
}
