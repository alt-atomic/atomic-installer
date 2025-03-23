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

package steps

import (
	"fmt"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"golang.org/x/text/language"
	"installer/app/image"
	"installer/app/utility"
	"installer/lib"
)

// Изначально chosenLangIndex = -1 означает, что выбор еще не задан
var chosenLangIndex int = -1

// CreateLanguageStep – шаг выбора языка.
func CreateLanguageStep(window *adw.ApplicationWindow, updateStep func(), onLanguageSelected func(string), onCancel func()) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconLanguage)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	box.Append(wrapper)

	// Список языков
	languages := []string{
		"Русский",
		"English",
	}

	combo := gtk.NewComboBoxText()
	combo.SetSizeRequest(300, -1)
	for _, lang := range languages {
		combo.AppendText(lang)
	}

	// Инициализировать выбранный язык по умолчанию только при первом запуске
	if chosenLangIndex == -1 {
		locale := utility.GetSystemLocale()
		switch locale {
		case language.English:
			chosenLangIndex = 1
		default:
			chosenLangIndex = 0
		}
	}
	combo.SetActive(chosenLangIndex)

	centerBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	centerBox.SetHExpand(true)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	centerBox.Append(combo)

	box.Append(centerBox)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	backBtn := gtk.NewButtonWithLabel(lib.T("Exit"))
	chooseBtn := gtk.NewButtonWithLabel(lib.T("Select"))

	backBtn.SetSizeRequest(120, 40)
	chooseBtn.SetSizeRequest(120, 40)

	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(backBtn)
	buttonBox.Append(chooseBtn)
	box.Append(buttonBox)

	parent := castToGtkWindow(window)

	combo.ConnectChanged(func() {
		activeIdx := combo.Active()
		if activeIdx < 0 {
			return
		}

		chosenLangIndex = activeIdx
		selectedLang := languages[activeIdx]
		var langCode string
		switch selectedLang {
		case "Русский":
			langCode = "ru"
		case "English":
			langCode = "en"
		default:
			langCode = "ru"
		}
		lib.SetLanguage(langCode)
		updateStep()
	})

	backBtn.ConnectClicked(func() {
		dialog := gtk.NewMessageDialog(
			parent,
			gtk.DialogModal,
			gtk.MessageQuestion,
			gtk.ButtonsNone,
		)
		dialog.SetTitle(lib.T("Installation"))
		dialog.Object.SetObjectProperty("secondary-text", lib.T("Do you really want to exit ?"))

		dialog.AddButton(lib.T("No"), int(gtk.ResponseCancel))
		dialog.AddButton(lib.T("Yes"), int(gtk.ResponseOK))

		dialog.ConnectResponse(func(responseID int) {
			if responseID == int(gtk.ResponseOK) {
				if onCancel != nil {
					onCancel()
				}
			}
			dialog.Destroy()
		})

		dialog.Show()
	})

	chooseBtn.ConnectClicked(func() {
		activeIdx := combo.Active()
		if activeIdx < 0 {
			fmt.Println("lang not selected")
			return
		}
		selectedLang := languages[activeIdx]
		onLanguageSelected(selectedLang)
	})

	return box
}
