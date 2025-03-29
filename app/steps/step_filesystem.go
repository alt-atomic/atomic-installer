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

// CreateFilesystemStep возвращает GUI-шаг выбора файловой системы.
func CreateFilesystemStep(onFsSelected func(string)) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconFilesystem)
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

	// Список вариантов (для ComboBoxText)
	fsChoices := []string{
		"btrfs subvolume:@, @home, @var",
		"ext4 ",
	}

	combo := gtk.NewComboBoxText()
	for _, choice := range fsChoices {
		combo.AppendText(choice)
	}
	combo.SetActive(0)
	centerBox.Append(combo)

	// Метка, которая будет показывать дополнительные описания
	noteLabel := gtk.NewLabel("")
	noteLabel.SetHAlign(gtk.AlignStart)
	noteLabel.SetMarginTop(10)
	noteLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(noteLabel)

	// Изначально для btrfs
	noteLabel.SetLabel(lib.T_("btrfs – recommended choice, works well with atomic image"))

	// Меняем описание при смене выбора
	combo.ConnectChanged(func() {
		activeIndex := combo.Active()
		if activeIndex < 0 {
			noteLabel.SetLabel("")
			return
		}
		if activeIndex == 0 {
			noteLabel.SetLabel(lib.T_("btrfs – recommended choice, works well with atomic image"))
		} else {
			noteLabel.SetLabel(lib.T_("ext4 – classic, proven file system"))
		}
	})

	// Горизонтальный контейнер для кнопок внизу
	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	chooseBtn := gtk.NewButtonWithLabel(lib.T_("Continue"))
	chooseBtn.SetSizeRequest(150, 45)
	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(chooseBtn)
	outerBox.Append(buttonBox)

	// Обработчик "Выбрать"
	chooseBtn.ConnectClicked(func() {
		activeIndex := combo.Active()
		if activeIndex < 0 {
			return
		}

		chosenStr := fsChoices[activeIndex]
		fsName := chosenStr
		if idx := strings.Index(fsName, " "); idx != -1 {
			fsName = fsName[:idx]
		}

		onFsSelected(fsName)
	})

	return outerBox
}
