package assets

import (
	_ "embed"
)

//go:embed icons/IconsFlat-32.json
var IconsAtlasJSON []byte

//go:embed icons/IconsFlat-32.png
var IconsAtlasPNG []byte

//go:embed icons/logo.png
var AppLogoPNG []byte
