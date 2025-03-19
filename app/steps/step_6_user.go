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
	"installer/app/image"
	"installer/lib"
	"regexp"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateUserStep – GUI-шаг для создания пользователя.
func CreateUserStep(onUserCreated func(string, string), onCancel func()) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconUser)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)

	wrapper.Append(pic)
	outerBox.Append(wrapper)

	contentBox := gtk.NewBox(gtk.OrientationVertical, 12)
	contentBox.SetVExpand(true) // Чтобы занять всё пространство
	outerBox.Append(contentBox)

	usernameLabel := gtk.NewLabel(fmt.Sprintf("%s:", lib.T("Login")))
	usernameLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(usernameLabel)

	usernameEntry := gtk.NewEntry()
	usernameEntry.SetPlaceholderText("username")
	usernameEntry.SetSizeRequest(250, -1)
	contentBox.Append(usernameEntry)

	passwordLabel := gtk.NewLabel(fmt.Sprintf("%s:", lib.T("Password")))
	passwordLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(passwordLabel)

	passwordEntry := gtk.NewEntry()
	passwordEntry.SetPlaceholderText("******")
	passwordEntry.SetVisibility(false)
	passwordEntry.SetInputPurpose(gtk.InputPurposePassword)
	passwordEntry.SetSizeRequest(250, -1)
	contentBox.Append(passwordEntry)

	repeatLabel := gtk.NewLabel(fmt.Sprintf("%s:", lib.T("Repeat password")))
	repeatLabel.SetHAlign(gtk.AlignStart)
	contentBox.Append(repeatLabel)

	repeatEntry := gtk.NewEntry()
	repeatEntry.SetPlaceholderText("******")
	repeatEntry.SetVisibility(false)
	repeatEntry.SetInputPurpose(gtk.InputPurposePassword)
	repeatEntry.SetSizeRequest(250, -1)
	contentBox.Append(repeatEntry)

	// Метка для вывода ошибок
	errorLabel := gtk.NewLabel("")
	errorLabel.SetHAlign(gtk.AlignStart)
	errorLabel.SetMarginTop(8)
	contentBox.Append(errorLabel)

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

	// Обработчик "Отмена"
	cancelBtn.ConnectClicked(func() {
		if onCancel != nil {
			onCancel()
		}
	})

	// В обработчике кнопки "Выбрать"
	chooseBtn.ConnectClicked(func() {
		userName := usernameEntry.Text()
		pass := passwordEntry.Text()
		passRepeat := repeatEntry.Text()

		// Проверка, что поля не пустые.
		if userName == "" || pass == "" {
			errorLabel.SetLabel(lib.T("Login and password cannot be empty"))
			return
		}

		// Проверка, что логин и пароль содержат только латинские символы и цифры.
		latinRegex := regexp.MustCompile(`^[A-Za-z0-9]+$`)
		if !latinRegex.MatchString(userName) {
			errorLabel.SetLabel(lib.T("The login must contain only Latin letters and numbers"))
			return
		}

		if pass != passRepeat {
			errorLabel.SetLabel(lib.T("Passwords do not match. Try again"))
			return
		}

		// Проверка, что логин не состоит только из цифр.
		isDigits, err := regexp.MatchString(`^\d+$`, userName)
		if err != nil {
			errorLabel.SetLabel(lib.T("Login verification error"))
			return
		}

		if isDigits {
			errorLabel.SetLabel(lib.T("Login cannot consist only of numbers"))
			return
		}

		onUserCreated(userName, pass)
	})

	return outerBox
}
