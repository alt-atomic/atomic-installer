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
	"installer/app/image"
	"installer/lib"
	"os"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateBootLoaderStep – GUI-шаг выбора загрузчика (UEFI или LEGACY).
func CreateBootLoaderStep(onBootModeSelected func(string), onCancel func()) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconBoot)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	outerBox.Append(wrapper)

	centerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	outerBox.Append(centerBox)

	// Проверяем поддержку UEFI
	uefiSupported := checkUEFISupport()

	var choices []string
	if uefiSupported {
		choices = []string{
			lib.T("UEFI (recommended for modern systems)"),
			lib.T("LEGACY (compatible variant)"),
		}
	} else {
		choices = []string{
			lib.T("LEGACY (UEFI not supported)"),
		}
	}

	combo := gtk.NewComboBoxText()
	for _, c := range choices {
		combo.AppendText(c)
	}
	combo.SetActive(0)
	centerBox.Append(combo)

	infoLabel := gtk.NewLabel("")
	infoLabel.SetHAlign(gtk.AlignStart)
	infoLabel.SetMarginTop(10)
	infoLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(infoLabel)

	// Начальный текст
	if uefiSupported {
		infoLabel.SetLabel(lib.T("Your computer supports UEFI boot - this is the recommended choice"))
	} else {
		infoLabel.SetLabel(lib.T("UEFI is not supported on this system, use LEGACY"))
	}

	// При смене выбора (если нужно динамически менять подсказку)
	combo.ConnectChanged(func() {
		idx := combo.Active()
		if !uefiSupported || idx < 0 {
			return
		}
		if idx == 0 {
			infoLabel.SetLabel(lib.T("UEFI mode is selected - this is the recommended choice for modern systems"))
		} else {
			infoLabel.SetLabel(lib.T("LEGACY is selected - a more compatible option for BIOS/UEFI"))
		}
	})

	// Горизонтальный контейнер для кнопок внизу
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	cancelBtn := gtk.NewButtonWithLabel(lib.T("Back"))
	chooseBtn := gtk.NewButtonWithLabel(lib.T("Select"))

	cancelBtn.SetSizeRequest(120, 40)
	chooseBtn.SetSizeRequest(120, 40)
	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(cancelBtn)
	buttonBox.Append(chooseBtn)
	outerBox.Append(buttonBox)

	// Обработка "Назад"
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	// «Выбрать»
	chooseBtn.ConnectClicked(func() {
		active := combo.Active()
		if active < 0 {
			return
		}
		chosenStr := choices[active]
		// Обрезаем "UEFI " или "LEGACY "
		if idx := strings.Index(chosenStr, " "); idx != -1 {
			chosenStr = chosenStr[:idx]
		}

		onBootModeSelected(chosenStr)
	})

	return outerBox
}

// checkUEFISupport – упрощённая проверка наличия каталога /sys/firmware/efi/efivars
func checkUEFISupport() bool {
	_, err := os.Stat("/sys/firmware/efi/efivars")
	return err == nil
}
