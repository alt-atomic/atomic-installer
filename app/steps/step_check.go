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
	"context"
	"fmt"
	"installer/app/image"
	"installer/app/utility"
	"installer/lib"
	"os/exec"
	"strings"
	"time"

	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

// CreateCheckDeviceStep - шаг приветствия и проверки устройства
func CreateCheckDeviceStep(onNext func()) gtk.Widgetter {
	box := gtk.NewBox(gtk.OrientationVertical, 12)
	box.SetMarginTop(20)
	box.SetMarginBottom(20)
	box.SetMarginStart(20)
	box.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconInstall)
	pic := iconWidget.(*gtk.Picture)
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(128, 128)
	wrapper.SetHAlign(gtk.AlignCenter)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)
	wrapper.Append(pic)
	box.Append(wrapper)

	centerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	centerBox.SetHAlign(gtk.AlignCenter)
	centerBox.SetVAlign(gtk.AlignCenter)
	centerBox.SetHExpand(true)
	centerBox.SetVExpand(true)
	spinner := gtk.NewSpinner()
	spinner.SetSizeRequest(30, 30)
	label := gtk.NewLabel(lib.T_("Checking device..."))
	label.SetUseMarkup(true)
	label.SetMarkup("<span size='xx-large'><b>" + lib.T_("Checking device...") + "</b></span>")
	centerBox.Append(spinner)
	centerBox.Append(label)

	box.Append(centerBox)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	chooseBtn := gtk.NewButtonWithLabel(lib.T_("Continue"))
	chooseBtn.SetSizeRequest(150, 45)
	chooseBtn.AddCSSClass("suggested-action")
	chooseBtn.SetSensitive(false)
	buttonBox.Append(chooseBtn)
	box.Append(buttonBox)

	spinner.Start()

	go func() {
		diskAvailable, err := checkDiskSize()
		if err != nil || !diskAvailable {
			glib.IdleAdd(func() bool {
				spinner.Stop()
				if err != nil {
					label.SetMarkup("<span size='xx-large'><b>" + err.Error() + "</b></span>")
				} else {
					label.SetMarkup("<span size='xx-large'><b>" + lib.T_("Insufficient disk space. At least 60GB required") + "</b></span>")
				}
				label.AddCSSClass("error")
				chooseBtn.SetSensitive(false)
				return false
			})
			return
		}

		firstRun := true
		for {
			if firstRun {
				time.Sleep(2 * time.Second)
				firstRun = false
			}
			connected := utility.CheckInternet()
			glib.IdleAdd(func() bool {
				if connected {
					spinner.Stop()
					label.SetMarkup("<span size='xx-large'><b>" + lib.T_("The device is ready for installation!") + "</b></span>")
					label.RemoveCSSClass("error")
					label.AddCSSClass("success")
					chooseBtn.SetSensitive(true)
				} else {
					spinner.Start()
					label.SetMarkup("<span size='xx-large'><b>" + lib.T_("Check your internet connection") + "</b></span>")
					label.AddCSSClass("error")
					label.RemoveCSSClass("success")
					chooseBtn.SetSensitive(false)
				}
				return false
			})
			if connected {
				break
			}
			time.Sleep(5 * time.Second)
		}
	}()

	chooseBtn.ConnectClicked(func() {
		if onNext != nil {
			onNext()
		}
	})

	return box
}

func checkDiskSize() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "lsblk", "-o", "NAME,SIZE,TYPE,MODEL", "-d", "-n")
	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("%s: %w", lib.T_("Error getting disk list"), err)
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	diskAvailable := false
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		sizeStr := fields[1]

		sizeGB, err := parseSize(sizeStr)
		if err != nil {
			continue
		}
		if sizeGB >= 60 {
			diskAvailable = true
			break
		}
	}

	return diskAvailable, nil
}
