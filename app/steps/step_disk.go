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
	"os/exec"
	"strconv"
	"strings"

	"github.com/diamondburned/gotk4/pkg/gtk/v4"
)

type DiskInfo struct {
	Path   string
	Size   string
	Model  string
	SizeGB float64
}

// CreateDiskStep – виджет для выбора диска
func CreateDiskStep(onDiskSelected func(string)) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	iconWidget := image.NewIconFromEmbed(image.IconDisk)
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

	// Получаем список дисков
	disks := getAvailableDisks()
	if len(disks) == 0 {
		lib.Log.Error("No suitable disks (≥ 60 GB) found.")
	}

	combo := gtk.NewComboBoxText()
	for _, d := range disks {
		display := fmt.Sprintf("%s (%s)", d.Path, d.Size)
		if d.Model != "" {
			display += " - " + d.Model
		}
		combo.AppendText(display)
	}
	if len(disks) > 0 {
		combo.SetActive(0)
	}
	centerBox.Append(combo)

	descLabel := gtk.NewLabel(fmt.Sprintf("%s ≥ 60 ГБ", lib.T_("Disk size")))
	descLabel.SetHAlign(gtk.AlignStart)
	descLabel.SetMarginTop(10)
	descLabel.SetHAlign(gtk.AlignCenter)
	centerBox.Append(descLabel)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)

	chooseBtn := gtk.NewButtonWithLabel(lib.T_("Continue"))
	chooseBtn.SetSizeRequest(150, 45)
	chooseBtn.AddCSSClass("suggested-action")

	buttonBox.Append(chooseBtn)
	outerBox.Append(buttonBox)

	chooseBtn.ConnectClicked(func() {
		active := combo.Active()
		if active < 0 {
			return
		}
		chosenDisk := disks[active].Path
		onDiskSelected(chosenDisk)
	})

	return outerBox
}

func getAvailableDisks() []DiskInfo {
	out, err := exec.Command("lsblk", "-o", "NAME,SIZE,TYPE,MODEL", "-d", "-n").Output()
	if err != nil {
		lib.Log.Error("Error getting disk list:", err)
		return nil
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	var result []DiskInfo

	for _, line := range lines {
		fields := strings.Fields(line)
		var name, sizeStr, devType, model string

		if len(fields) == 3 {
			name, sizeStr, devType = fields[0], fields[1], fields[2]
			model = lib.T_("unknown model")
		} else if len(fields) >= 4 {
			name, sizeStr, devType, model = fields[0], fields[1], fields[2], fields[3]
		}

		if devType != "disk" {
			continue
		}
		if strings.HasPrefix(name, "zram") || strings.HasPrefix(name, "loop") {
			continue
		}
		sizeGB, err := parseSize(sizeStr)
		if err != nil {
			continue
		}
		if sizeGB < 60 {
			continue
		}
		path := "/dev/" + name
		info := DiskInfo{
			Path:   path,
			Size:   sizeStr,
			Model:  model,
			SizeGB: sizeGB,
		}
		result = append(result, info)
	}
	return result
}

func parseSize(sizeStr string) (float64, error) {
	if len(sizeStr) < 2 {
		return 0, fmt.Errorf("unknown size format: %s", sizeStr)
	}
	sizeStr = strings.ReplaceAll(sizeStr, ",", ".")
	unit := sizeStr[len(sizeStr)-1]
	valStr := sizeStr[:len(sizeStr)-1]
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		return 0, fmt.Errorf("error parsing number from %s: %w", valStr, err)
	}
	switch unit {
	case 'G':
		return val, nil
	case 'M':
		return val / 1024.0, nil
	case 'T':
		return val * 1024.0, nil
	default:
		return 0, fmt.Errorf("unknown unit: %c", unit)
	}
}
