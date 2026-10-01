// Package assets embeds decoded native game data and YM soundtracks.
package assets

import "embed"

//go:embed *.bin *.json *.png sprites/*
var Files embed.FS
