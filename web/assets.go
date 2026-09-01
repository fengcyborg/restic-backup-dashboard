package web

import (
	"embed"
	"io/fs"
)

// Assets contains the dashboard UI so releases can be distributed as one binary.
//
//go:embed static/*
var assets embed.FS

func Assets() (fs.FS, error) {
	return fs.Sub(assets, "static")
}
