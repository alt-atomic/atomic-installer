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
	"bytes"
	"errors"
	"fmt"
	"installer/app/image"
	"installer/lib"
	"os/exec"
	"strings"
	"sync"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// ImagePodman – структура для парсинга "podman images --format json"
type ImagePodman struct {
	Names []string `json:"Names"`
}

// Choice – элемент списка доступных образов
type Choice struct {
	Name        string
	ShortText   string
	Description string
}

// getAvailableImages – заглушка вместо реального podman.
func getAvailableImages() []Choice {
	var images []Choice
	return addDefaultImage(images)
}

// addDefaultImage – добавляет «стандартные» образы
func addDefaultImage(images []Choice) []Choice {
	if images == nil {
		images = []Choice{}
	}
	images = append(
		images,
		Choice{
			Name:        "ghcr.io/alt-gnome/alt-atomic:latest",
			ShortText:   "GNOME",
			Description: lib.T_("GNOME Image. Recommended"),
		},
		Choice{
			Name:        "ghcr.io/alt-gnome/alt-atomic:latest-nv",
			ShortText:   "GNOME NVIDIA",
			Description: lib.T_("GNOME image for NVIDIA. OPEN driver"),
		},
		Choice{
			Name:        "ghcr.io/alt-atomic/alt-kde:latest",
			ShortText:   "KDE",
			Description: lib.T_("KDE image. In testing phase, not recommended"),
		},
		Choice{
			Name:        "ghcr.io/alt-atomic/alt-kde:latest-nv",
			ShortText:   "KDE NVIDIA",
			Description: lib.T_("KDE image for NVIDIA. In testing, not recommended"),
		},
	)
	return images
}

// validateImage – проверяем образ через `skopeo inspect`.
func validateImage(image string) (string, error) {
	cmd := exec.Command("skopeo", "inspect", "docker://"+image)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	output, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return stderr.String(), err
		}
		return lib.T_("Error executing command (check that skopeo is installed)"), err
	}
	return string(output), nil
}

// CreateImageStep – виджет для шага выбора образа.
func CreateImageStep(onImageSelected func(string)) gtk.Widgetter {
	// ВЕРТИКАЛЬНЫЙ box – «корневой»
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)

	iconWidget := image.NewIconFromEmbed(image.IconImage)
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

	// Получаем список «стандартных» образов
	images := getAvailableImages()

	// Добавляем пункт «кастомный» (последним)
	images = append(images, Choice{
		Name:        lib.T_("Add your image"),
		Description: "",
	})

	customChoiceIndex := len(images) - 1

	combo := gtk.NewComboBoxText()
	var comboCount = len(images)

	for _, img := range images {
		label := img.Name
		if img.ShortText != "" {
			label += "  " + img.ShortText
		}
		combo.AppendText(label)
	}
	combo.SetActive(0)
	centerBox.Append(combo)

	// Лейбл для описания
	descLabel := gtk.NewLabel("")
	descLabel.SetHAlign(gtk.AlignStart)
	descLabel.SetMarginTop(10)
	descLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(descLabel)

	// Поле для ввода кастомного образа (по умолчанию скрыто)
	customEntry := gtk.NewEntry()
	customEntry.SetPlaceholderText(lib.T_("Enter the image link"))
	customEntry.SetVisible(false)
	centerBox.Append(customEntry)

	// Кнопка "Проверить и добавить" + Спиннер
	checkButton := gtk.NewButtonWithLabel(lib.T_("Check and add"))
	checkButton.SetVisible(false)

	spinner := gtk.NewSpinner()
	spinner.SetHAlign(gtk.AlignCenter)
	spinner.SetVAlign(gtk.AlignCenter)

	stack := gtk.NewStack()
	stack.AddNamed(checkButton, "button")
	stack.AddNamed(spinner, "spinner")
	stack.SetVisibleChildName("button")
	centerBox.Append(stack)

	// Лейбл для результата проверки
	checkResultLabel := gtk.NewLabel("")
	checkResultLabel.SetHAlign(gtk.AlignStart)
	checkResultLabel.SetMarginTop(4)
	checkResultLabel.SetVisible(false)
	centerBox.Append(checkResultLabel)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	chooseBtn := gtk.NewButtonWithLabel(lib.T_("Continue"))
	chooseBtn.SetSizeRequest(150, 45)
	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(chooseBtn)

	outerBox.Append(buttonBox)

	var customImageValid string

	// При смене пункта в combo
	combo.ConnectChanged(func() {
		checkResultLabel.SetVisible(false)
		checkResultLabel.SetLabel("")
		customEntry.SetText("")
		customImageValid = ""

		activeIndex := combo.Active()
		if activeIndex < 0 {
			return
		}

		if activeIndex == customChoiceIndex {
			// Кастомный пункт
			customEntry.SetVisible(true)
			checkButton.SetVisible(true)
			stack.SetVisibleChildName("button")
			descLabel.SetLabel(lib.T_("Enter your image manually and check"))
		} else {
			// Стандартный пункт
			customEntry.SetVisible(false)
			checkButton.SetVisible(false)
			stack.SetVisibleChildName("button")

			desc := images[activeIndex].Description
			if desc == "" {
				desc = lib.T_("Empty description")
			}
			descLabel.SetLabel(desc)
		}
	})

	// Нажали кнопку "Проверить и добавить"
	checkButton.ConnectClicked(func() {
		imageName := strings.TrimSpace(customEntry.Text())
		if imageName == "" {
			checkResultLabel.SetLabel(lib.T_("Please enter a valid image name"))
			checkResultLabel.SetVisible(true)
			return
		}

		// Переходим на spinner
		stack.SetVisibleChildName("spinner")
		spinner.Start()

		checkButton.SetSensitive(false)
		chooseBtn.SetSensitive(false)

		// Запускаем проверку в горутине
		go func(img string) {
			var mu sync.Mutex
			mu.Lock()
			out, err := validateImage(img)
			mu.Unlock()

			// Возврат в UI-поток
			glib.IdleAdd(func() bool {
				spinner.Stop()
				stack.SetVisibleChildName("button")
				checkButton.SetSensitive(true)
				chooseBtn.SetSensitive(true)

				if err != nil {
					checkResultLabel.SetLabel(fmt.Sprintf("%s :\n %s", lib.T_("Image verification error"), out))
					checkResultLabel.SetVisible(true)
					checkResultLabel.AddCSSClass("error")
					customImageValid = ""
				} else {
					checkResultLabel.SetLabel(lib.T_("The image has been verified and added to the list"))
					checkResultLabel.SetVisible(true)
					checkResultLabel.RemoveCSSClass("error")
					customImageValid = imageName

					images = append(images, Choice{
						Name:        imageName,
						Description: "",
					})

					combo.AppendText(imageName)
					comboCount++
					combo.SetActive(comboCount - 1)
				}
				return false
			})
		}(imageName)
	})

	// Нажали «Выбрать»
	chooseBtn.ConnectClicked(func() {
		activeIndex := combo.Active()
		if activeIndex < 0 {
			return
		}

		var resultImage string
		if activeIndex == customChoiceIndex {
			if customImageValid == "" {
				checkResultLabel.SetLabel(lib.T_("First check the entered image"))
				checkResultLabel.SetVisible(true)
				return
			}
			resultImage = customImageValid
		} else if activeIndex >= len(images) {
			resultImage = images[activeIndex].Name
		} else {
			resultImage = images[activeIndex].Name
		}

		onImageSelected(resultImage)
	})

	// Изначальное описание (для пункта 0)
	if combo.Active() == 0 {
		desc := images[0].Description
		if desc == "" {
			descLabel.SetLabel(lib.T_("Empty description"))
		} else {
			descLabel.SetLabel(desc)
		}
	}

	return outerBox
}
