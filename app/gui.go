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

package app

import (
	"fmt"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"installer/app/steps"
	"installer/lib"
	"os"
	"unsafe"
)

// InstallerViewService — сервис
type InstallerViewService struct{}

// NewInstallerViewService — конструктор сервиса
func NewInstallerViewService() *InstallerViewService {
	return &InstallerViewService{}
}

func (i *InstallerViewService) getStepTitles() []string {
	return []string{
		fmt.Sprintf("1. %s", lib.T_("Language selection")),
		fmt.Sprintf("2. %s", lib.T_("Device check")),
		fmt.Sprintf("2. %s", lib.T_("Image selection")),
		fmt.Sprintf("3. %s", lib.T_("Disk selection")),
		fmt.Sprintf("4. %s", lib.T_("Filesystem selection")),
		fmt.Sprintf("5. %s", lib.T_("Bootloader selection")),
		fmt.Sprintf("6. %s", lib.T_("User selection")),
		fmt.Sprintf("7. %s", lib.T_("Summary")),
		fmt.Sprintf("8. %s", lib.T_("Installation")),
	}
}

// OnActivate - главный цикл приложения
func (i *InstallerViewService) OnActivate(app *adw.Application) {
	window := i.NewAdwApplicationWindow(app)
	window.SetDefaultSize(900, 700)
	window.SetTitle(lib.T_("Installation"))
	var currentStep int

	stepsCount := len(i.getStepTitles())

	// Если последний шаг, блокируем закрытие приложения
	window.ConnectCloseRequest(func() bool {
		if currentStep == stepsCount-1 {
			lib.Log.Warning("exit blocked, installation started")
			return true
		}

		return false
	})

	toolbarView := adw.NewToolbarView()

	mainHeader := adw.NewHeaderBar()
	boldLabel := gtk.NewLabel("")
	boldLabel.SetUseMarkup(true)
	boldLabel.SetLabel("<b>Atomic Installer</b>")
	mainHeader.SetTitleWidget(boldLabel)
	toolbarView.AddTopBar(mainHeader)

	navCenterBox := gtk.NewCenterBox()

	backBtn := gtk.NewButton()
	backIcon := gtk.NewImageFromIconName("go-previous-symbolic")
	backIcon.SetPixelSize(22)
	backBtn.SetChild(backIcon)
	backBtn.AddCSSClass("circular")
	backBtn.AddCSSClass("flat")

	nextBtn := gtk.NewButton()
	nextIcon := gtk.NewImageFromIconName("go-next-symbolic")
	nextIcon.SetPixelSize(22)
	nextBtn.SetChild(nextIcon)
	nextBtn.AddCSSClass("circular")
	nextBtn.AddCSSClass("flat")
	stepLabel := gtk.NewLabel("")

	leftBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	leftBox.SetMarginStart(20)
	leftBox.Append(backBtn)

	rightBox := gtk.NewBox(gtk.OrientationHorizontal, 0)
	rightBox.SetMarginEnd(20)
	rightBox.Append(nextBtn)

	navCenterBox.SetStartWidget(leftBox)
	navCenterBox.SetCenterWidget(stepLabel)
	navCenterBox.SetEndWidget(rightBox)
	toolbarView.AddTopBar(navCenterBox)

	var chosenImage string
	var chosenDisk string
	var chosenFilesystem string
	var chosenBootMode string
	var chosenUsername string
	var chosenPassword string
	var chosenLang string

	stepDone := make([]bool, stepsCount)

	var stepsArr []func() gtk.Widgetter

	// Функция, которая будет обновлять отображение контента
	updateStep := func() {
		stepLabel.SetLabel(i.getStepTitles()[currentStep])

		// Создаём новый box
		newContent := gtk.NewBox(gtk.OrientationVertical, 10)
		newContent.SetMarginTop(20)
		newContent.SetMarginBottom(20)
		newContent.SetMarginStart(20)
		newContent.SetMarginEnd(20)

		// Генерируем виджет для текущего шага
		stepWidget := stepsArr[currentStep]()
		newContent.Append(stepWidget)
		toolbarView.SetContent(newContent)

		// «Назад» доступен, если это не первый шаг
		backBtn.SetSensitive(currentStep > 0)

		// «Вперёд» доступен, только если этот шаг уже завершён
		nextBtn.SetSensitive(stepDone[currentStep])

		if currentStep == stepsCount-1 {
			backBtn.SetSensitive(false)
			nextBtn.SetSensitive(false)
			nextBtn.SetTooltipText(lib.T_("Waiting for installation to complete"))
			nextBtn.SetTooltipText(lib.T_("Ready"))
		} else {
			backBtn.SetSensitive(currentStep > 0)
			nextBtn.SetSensitive(stepDone[currentStep])
			nextBtn.SetTooltipText(lib.T_("Next"))
		}
	}

	// Заполняем stepsArr
	stepsArr = []func() gtk.Widgetter{
		// Шаг 0: Выбор языка
		func() gtk.Widgetter {
			return steps.CreateLanguageStep(
				func() {
					updateStep()
				},
				func(lang string) {
					chosenLang = lang
					stepDone[0] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 1: Проверка устройства
		func() gtk.Widgetter {
			return steps.CreateCheckDeviceStep(
				func() {
					updateStep()
					stepDone[1] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 2: Выбор образа
		func() gtk.Widgetter {
			return steps.CreateImageStep(
				func(selected string) {
					chosenImage = selected
					stepDone[2] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 3: Выбор диска
		func() gtk.Widgetter {
			return steps.CreateDiskStep(
				func(disk string) {
					chosenDisk = disk
					stepDone[3] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 4: Выбор файловой системы
		func() gtk.Widgetter {
			return steps.CreateFilesystemStep(
				func(fs string) {
					chosenFilesystem = fs
					stepDone[4] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 5: Выбор загрузчика
		func() gtk.Widgetter {
			return steps.CreateBootLoaderStep(
				func(bootMode string) {
					chosenBootMode = bootMode
					stepDone[5] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 6: Создание пользователя
		func() gtk.Widgetter {
			return steps.CreateUserStep(
				func(username, password string) {
					chosenUsername = username
					chosenPassword = password
					stepDone[6] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 7: Итог
		func() gtk.Widgetter {
			return steps.CreateSummaryStep(
				chosenLang,
				chosenImage,
				chosenDisk,
				chosenFilesystem,
				chosenBootMode,
				chosenUsername,
				chosenPassword,
				func() {
					stepDone[7] = true
					nextBtn.SetSensitive(true)
					currentStep++
					updateStep()
				},
			)
		},
		// Шаг 7: Установка
		func() gtk.Widgetter {
			return steps.CreateInstallProgressStep(
				window,
				chosenLang,
				chosenImage,
				chosenDisk,
				chosenFilesystem,
				chosenBootMode,
				chosenUsername,
				chosenPassword,
				func() {
					os.Exit(0)
				},
			)
		},
	}

	// Изначально 0-й шаг, он не завершён
	currentStep = 0
	stepDone[0] = false

	for i := 1; i < stepsCount; i++ {
		stepDone[i] = false
	}

	updateStep()

	// Кнопка «Назад»
	backBtn.ConnectClicked(func() {
		if currentStep > 0 {
			currentStep--
			updateStep()
		}
	})

	// Кнопка «Вперёд»
	nextBtn.ConnectClicked(func() {
		if currentStep < stepsCount-1 {
			currentStep++
			updateStep()
		} else {
			window.Close()
		}
	})

	window.SetContent(toolbarView)
	window.SetVisible(true)
}

func (i *InstallerViewService) NewAdwApplicationWindow(app *adw.Application) *adw.ApplicationWindow {
	gtkApp := (*gtk.Application)(unsafe.Pointer(app))
	return adw.NewApplicationWindow(gtkApp)
}
