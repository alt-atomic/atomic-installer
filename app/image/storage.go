// Atomic Installer
// Copyright (C) 2025 Дмитрий Удалов dmitry@udalov.online
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package image

import (
	_ "embed" // Включаем поддержку go:embed
	"log"

	"github.com/diamondburned/gotk4/pkg/gdk/v4"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

//go:embed icons/translate.png
var iconLanguage []byte

//go:embed icons/docker.png
var iconImage []byte

//go:embed icons/disk.png
var iconDisk []byte

//go:embed icons/folder.png
var iconFilesystem []byte

//go:embed icons/loader.png
var iconLoader []byte

//go:embed icons/user.png
var iconUser []byte

//go:embed icons/layers.png
var iconResult []byte

//go:embed icons/analysis.png
var iconInstall []byte

const (
	IconLanguage = iota
	IconImage
	IconDisk
	IconFilesystem
	IconBoot
	IconUser
	IconResult
	IconInstall
)

// NewIconFromEmbed возвращает готовый к вставке в UI виджет
func NewIconFromEmbed(iconType int) gtk.Widgetter {
	var icon []byte
	switch iconType {
	case IconInstall:
		icon = iconInstall
	case IconLanguage:
		icon = iconLanguage
	case IconImage:
		icon = iconImage
	case IconDisk:
		icon = iconDisk
	case IconFilesystem:
		icon = iconFilesystem
	case IconBoot:
		icon = iconLoader
	case IconUser:
		icon = iconUser
	case IconResult:
		icon = iconResult
	default:
		return nil
	}

	glibBytes := glib.NewBytesWithGo(icon)
	texture, err := gdk.NewTextureFromBytes(glibBytes)
	if err != nil {
		log.Println("Ошибка создания gdk.Texture из байтов:", err)
		return gtk.NewPictureForPaintable(nil)
	}

	pic := gtk.NewPictureForPaintable(texture)
	return pic
}
