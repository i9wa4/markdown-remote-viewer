package assets

import "embed"

// FS contains static files served by the viewer.
//
//go:embed static/*
var FS embed.FS
