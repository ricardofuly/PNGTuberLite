package assets

import (
	_ "embed"
)

//go:embed icons/add.png
var IconAddPNG []byte

//go:embed icons/audio.png
var IconAudioPNG []byte

//go:embed icons/avatar.png
var IconAvatarPNG []byte

//go:embed icons/change.png
var IconChangePNG []byte

//go:embed icons/close.png
var IconClosePNG []byte

//go:embed icons/delete.png
var IconDeletePNG []byte

//go:embed icons/duplicate.png
var IconDuplicatePNG []byte

//go:embed icons/editor.png
var IconEditorPNG []byte

//go:embed icons/enable.png
var IconEnablePNG []byte

//go:embed icons/favorite.png
var IconFavoritePNG []byte

//go:embed icons/fisica.png
var IconPhysicsPNG []byte

//go:embed icons/keys.png
var IconKeysPNG []byte

//go:embed icons/logo-256x256.png
var AppLogoPNG []byte

//go:embed icons/logo-32x32.png
var AppTrayPNG []byte

var AppIconICO = AppTrayPNG

//go:embed icons/obs.png
var IconOBSPNG []byte

//go:embed icons/open-editor.png
var IconOpenEditorPNG []byte

//go:embed icons/png-file.png
var IconPNGFilePNG []byte

//go:embed icons/remove.png
var IconRemovePNG []byte

//go:embed icons/restart.png
var IconRestartPNG []byte

//go:embed icons/restore.png
var IconRestorePNG []byte

//go:embed icons/roupas.png
var IconCostumesPNG []byte

//go:embed icons/save.png
var IconSavePNG []byte

//go:embed icons/selected.png
var IconSelectedPNG []byte

//go:embed icons/settings.png
var IconSettingsPNG []byte

//go:embed icons/update.png
var IconUpdatePNG []byte

//go:embed fonts/font.ttf
var RegularFontTTF []byte

//go:embed fonts/font_bold.ttf
var BoldFontTTF []byte

//go:embed samples/defaultAvatar.save
var DefaultAvatarSave []byte

//go:embed samples/slugcat.save
var SlugcatAvatarSave []byte
