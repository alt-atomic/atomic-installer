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

package lib

import (
	"fmt"
	"golang.org/x/text/language"
	"os"

	"github.com/leonelquinteros/gotext"
)

// InitLocales initializes translations using gotext.
func InitLocales() {
	if _, err := os.Stat(Env.PathLocales); os.IsNotExist(err) {
		textError := fmt.Sprintf("Translations folder not found at path: %s", Env.PathLocales)
		Log.Error(textError)
		panic(err)
	}

	gotext.Configure(Env.PathLocales, Env.Language.String(), "installer")

	Log.Info("Translations successfully initialized")
}

// SetLanguage changes the current language and reloads translations.
func SetLanguage(lang string) {
	newLang, err := language.Parse(lang)
	if err != nil {
		Log.Error(fmt.Sprintf("Error parsing language '%s': %v", lang, err))
		return
	}
	Env.Language = newLang
	gotext.Configure(Env.PathLocales, Env.Language.String(), "installer")
	Log.Info(fmt.Sprintf("Language switched to: %s", lang))
}

// T returns the translated string for the given message ID.
func T(messageID string) string {
	return gotext.Get(messageID)
}
