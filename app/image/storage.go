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
	"github.com/diamondburned/gotk4/pkg/gdkpixbuf/v2"
	"installer/lib"
	"log"
	"time"

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

//go:embed icons/install.png
var iconInstall []byte

//go:embed icons/animation.gif
var animGIF []byte

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

// NewAnimatedGifWidget возвращает виджет *gtk.Image с анимированным GIF.
func NewAnimatedGifWidget() gtk.Widgetter {
	loader := gdkpixbuf.NewPixbufLoader()
	if err := loader.Write(animGIF); err != nil {
		log.Println("Error writing GIF to PixbufLoader:", err)
		return gtk.NewLabel("Failed to load animated GIF.")
	}
	loader.Close()

	anim := loader.Animation()
	if anim == nil {
		return gtk.NewLabel("Failed to interpret GIF as animation.")
	}

	iter := anim.Iter(nil)
	if iter == nil {
		return gtk.NewLabel("Failed to get animation iterator.")
	}

	// Take the first frame
	firstPixbuf := iter.Pixbuf()
	if firstPixbuf == nil {
		return gtk.NewLabel("Failed to get the first frame of GIF.")
	}

	img := gtk.NewImageFromPixbuf(firstPixbuf)
	img.SetSizeRequest(150, 150)
	img.SetHExpand(true)
	img.SetVExpand(true)
	img.SetHAlign(gtk.AlignCenter)
	img.SetVAlign(gtk.AlignCenter)
	go animateGIF(anim, img)

	return img
}

// animateGIF – в фоне крутит кадры анимированного GIF и обновляет картинку.
func animateGIF(anim *gdkpixbuf.PixbufAnimation, img *gtk.Image) {
	// У каждого потока/циклической анимации должен быть свой итератор
	iter := anim.Iter(nil)
	if iter == nil {
		lib.Log.Error("Error: Failed to get animation iterator")
		return
	}

	for {
		delay := iter.DelayTime()
		if delay < 1 {
			delay = 100
		}

		time.Sleep(time.Duration(delay) * time.Millisecond)
		ok := iter.Advance(nil)
		if !ok {
			iter = anim.Iter(nil)
			if iter == nil {
				return
			}
		}
		pix := iter.Pixbuf()
		if pix == nil {
			continue
		}

		glib.IdleAdd(func() {
			// Масштабируем каждый кадр:
			scaled := pix.ScaleSimple(150, 150, gdkpixbuf.InterpBilinear)
			if scaled != nil {
				img.SetFromPixbuf(scaled)
			} else {
				img.SetFromPixbuf(pix)
			}
		})
	}
}
