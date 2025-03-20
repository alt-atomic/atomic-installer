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

package main

import (
	"fmt"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/gio/v2"
	"installer/app"
	"installer/app/utility"
	"installer/lib"
	"log"
	"os"
	"os/exec"
	"syscall"
)

func main() {
	checkRoot()
	lib.Env.Language = utility.GetSystemLocale()
	lib.InitConfig()
	lib.InitLocales()
	lib.InitLogger()

	err := checkCommands()
	if err != nil {
		log.Fatal(err)
	}

	serviceInstallerView := app.NewInstallerViewService()
	application := adw.NewApplication("com.example.AdwExampleApp", gio.ApplicationFlagsNone)
	application.ConnectActivate(func() {
		serviceInstallerView.OnActivate(application)
	})
	os.Exit(application.Run(os.Args))
}

// checkRoot проверка root прав
func checkRoot() {
	if syscall.Geteuid() != 0 {
		log.Println("The installer must be run with superuser (root)")
		os.Exit(1)
	}
}

// checkCommands проверяет наличие необходимых системных команд
func checkCommands() error {
	err := os.Setenv("PATH", os.Getenv("PATH")+":/usr/sbin:/sbin")
	if err != nil {
		return err
	}

	commands := []string{
		"podman",
		"rsync",
		"wipefs",
		"parted",
		"mkfs.fat",
		"mkfs.btrfs",
		"mkfs.ext4",
		"mount",
		"umount",
		"blkid",
		"lsblk",
	}
	for _, cmd := range commands {
		if _, err = exec.LookPath(cmd); err != nil {
			return fmt.Errorf("command %s not found in PATH", cmd)
		}
	}
	return nil
}
