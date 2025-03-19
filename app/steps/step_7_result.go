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
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateSummaryStep – финальный шаг, отображающий все выбранные параметры.
func CreateSummaryStep(
	chosenLang, chosenImage, chosenDisk, chosenFilesystem, chosenBootMode, chosenUsername, chosenPassword string,
	onInstall func(),
	onCancel func(),
) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconResult)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)
	wrapper.Append(pic)
	outerBox.Append(wrapper)

	centerBox := gtk.NewBox(gtk.OrientationVertical, 8)
	centerBox.SetVExpand(true)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	outerBox.Append(centerBox)

	grid := gtk.NewGrid()
	grid.SetColumnSpacing(12)
	grid.SetRowSpacing(4)
	centerBox.Append(grid)

	var row int
	addRow := func(field, value string) {
		lblField := gtk.NewLabel(field + ":")
		lblField.SetHAlign(gtk.AlignEnd)

		lblValue := gtk.NewLabel("")
		lblValue.SetUseMarkup(true)
		lblValue.SetLabel("<b>" + value + "</b>")
		lblValue.SetHAlign(gtk.AlignStart)

		// Размещаем в сетке: (столбец=0,row=текущаяСтрока), (столбец=1,row=текущаяСтрока)
		grid.Attach(lblField, 0, row, 1, 1)
		grid.Attach(lblValue, 1, row, 1, 1)

		row++
	}

	stars := strings.Repeat("*", len(chosenPassword))
	addRow(lib.T("User"), chosenUsername)
	addRow(lib.T("Password"), stars)
	addRow(lib.T("Bootloader"), chosenBootMode)
	addRow(lib.T("Selected image"), chosenImage)
	addRow(lib.T("System language"), chosenLang)
	addRow(lib.T("Selected disk"), chosenDisk)
	addRow(lib.T("Filesystem"), chosenFilesystem)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	cancelBtn := gtk.NewButtonWithLabel(lib.T("Back"))
	installBtn := gtk.NewButtonWithLabel(lib.T("Start install"))

	cancelBtn.SetSizeRequest(120, 40)
	installBtn.SetSizeRequest(160, 40)
	installBtn.AddCSSClass("suggested-action")

	buttonBox.Append(cancelBtn)
	buttonBox.Append(installBtn)
	outerBox.Append(buttonBox)

	// Обработчики
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	installBtn.ConnectClicked(func() {
		if onInstall != nil {
			onInstall()
		}
	})

	return outerBox
}
