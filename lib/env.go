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
	"github.com/ilyakaznacheev/cleanenv"
	"golang.org/x/text/language"
	"os"
	"path/filepath"
)

type Environment struct {
	PathLocales string `yaml:"pathLocales"`
	PathLogFile string `yaml:"pathLogFile"`
	Language    language.Tag
}

var Env Environment

// Глобальные переменные для возможности переопределения значений при сборке
var (
	BuildPathLocales string
	BuildPathLogFile string
)

func InitConfig() {
	var configPath string

	if BuildPathLocales != "" {
		Env.PathLocales = BuildPathLocales
	}
	if BuildPathLogFile != "" {
		Env.PathLogFile = BuildPathLogFile
	}

	// Ищем конфигурационный файл в текущей директории
	if _, err := os.Stat("config.yml"); err == nil {
		configPath = "config.yml"
	} else if _, err = os.Stat("/etc/apm/config.yml"); err == nil {
		configPath = "/etc/apm/config.yml"
	}

	// Если найден конфигурационный файл, читаем его
	if configPath != "" {
		err := cleanenv.ReadConfig(configPath, &Env)
		if err != nil {
			Log.Fatal(err)
		}
	}

	// Проверяем и создаём путь для лог-файла
	if err := EnsurePath(Env.PathLogFile); err != nil {
		Log.Fatal(err)
	}
}

// EnsurePath проверяет, существует ли файл и создает его при необходимости.
func EnsurePath(path string) error {
	dir := filepath.Dir(path)

	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		var file *os.File
		file, err = os.Create(path)
		if err != nil {
			return err
		}
		err = file.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

// EnsureDir проверяет, существует ли директория по указанному пути, и создает её при необходимости.
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
