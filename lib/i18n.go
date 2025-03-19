package lib

import (
	"fmt"
	"golang.org/x/text/language"
	"os"
	"path/filepath"

	"github.com/leonelquinteros/gotext"
)

var Locale *gotext.Locale

// InitLocales initializes translations using gotext.
func InitLocales() {
	if _, err := os.Stat(Env.PathLocales); os.IsNotExist(err) {
		textError := fmt.Sprintf("Translations folder not found at path: %s", Env.PathLocales)
		Log.Error(textError)
		panic(err)
	}

	gotext.Configure(Env.PathLocales, Env.Language.String(), "default")
	Locale = gotext.NewLocale(Env.PathLocales, Env.Language.String())

	localeFile := filepath.Join(Env.PathLocales, Env.Language.String(), "LC_MESSAGES", "default.po")
	if _, err := os.Stat(localeFile); err != nil {
		Log.Warn(fmt.Sprintf("Translation file not found: %s", localeFile))
	} else {
		Log.Info(fmt.Sprintf("Translation file loaded: %s", localeFile))
	}

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
	gotext.Configure(Env.PathLocales, Env.Language.String(), "default")
	Locale = gotext.NewLocale(Env.PathLocales, Env.Language.String())
	Log.Info(fmt.Sprintf("Language switched to: %s", lang))
}

// T returns the translated string for the given message ID.
func T(messageID string) string {
	translation := gotext.Get(messageID)
	if translation == messageID {
		return translation
	}

	return translation
}
