package steps

import (
	"fmt"
	"github.com/diamondburned/gotk4-adwaita/pkg/adw"
	"github.com/diamondburned/gotk4/pkg/glib/v2"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"installer/app/image"
	"installer/app/install"
	"installer/lib"
	"os/exec"
	"regexp"
	"strings"
)

var statusLabel *gtk.Label
var logView *gtk.TextView

// CreateInstallProgressStep – шаг, запускающий и показывающий процесс установки.
func CreateInstallProgressStep(window *adw.ApplicationWindow, chosenLang, chosenImage, chosenDisk, chosenFilesystem, chosenBootMode, chosenUsername, chosenPassword string, onCancel func()) gtk.Widgetter {
	outerBox := gtk.NewBox(gtk.OrientationVertical, 12)
	outerBox.SetMarginTop(20)
	outerBox.SetMarginBottom(20)
	outerBox.SetMarginStart(20)
	outerBox.SetMarginEnd(20)

	animWidget := image.NewAnimatedGifWidget()
	wrapper := gtk.NewBox(gtk.OrientationHorizontal, 0)
	wrapper.SetSizeRequest(150, 150)
	wrapper.SetHExpand(false)
	wrapper.SetVExpand(false)
	wrapper.Append(animWidget)
	outerBox.Append(wrapper)

	statusLabel = gtk.NewLabel("")
	statusLabel.SetUseMarkup(true)
	statusLabel.SetLabel(fmt.Sprintf("<big><b>%s</b></big>", lib.T("Start installation")))
	statusLabel.SetHAlign(gtk.AlignCenter)
	statusLabel.SetVAlign(gtk.AlignStart)
	outerBox.Append(statusLabel)

	scrolledWindow := gtk.NewScrolledWindow()
	scrolledWindow.SetHExpand(true)
	scrolledWindow.SetVExpand(true)

	logView = gtk.NewTextView()
	logView.SetWrapMode(gtk.WrapWordChar)
	logView.SetEditable(false)
	logView.SetCursorVisible(false)
	logView.Object.SetObjectProperty("left-margin", 10)

	scrolledWindow.SetChild(logView)
	outerBox.Append(scrolledWindow)

	buttonBox := gtk.NewBox(gtk.OrientationHorizontal, 20)
	buttonBox.SetHAlign(gtk.AlignCenter)
	buttonBox.SetMarginTop(20)
	outerBox.Append(buttonBox)

	cancelBtn := gtk.NewButtonWithLabel(lib.T("Cancel"))
	cancelBtn.SetSizeRequest(120, 40)
	buttonBox.Append(cancelBtn)

	parent := castToGtkWindow(window)

	// Обработчик для отмены установки (до завершения установки)
	cancelBtn.ConnectClicked(func() {
		dialog := gtk.NewMessageDialog(
			parent,
			gtk.DialogModal,
			gtk.MessageQuestion,
			gtk.ButtonsNone,
		)
		dialog.SetTitle(lib.T("Installation"))
		dialog.Object.SetObjectProperty("secondary-text", lib.T("Are you sure you want to cancel the installation ?"))

		dialog.AddButton(lib.T("No"), int(gtk.ResponseCancel))
		dialog.AddButton(lib.T("Yes"), int(gtk.ResponseOK))

		dialog.ConnectResponse(func(responseID int) {
			if responseID == int(gtk.ResponseOK) {
				if onCancel != nil {
					onCancel()
				}
			}
			dialog.Destroy()
		})
		dialog.Show()
	})

	user := install.User{
		Login:    chosenUsername,
		Password: chosenPassword,
	}

	installData := install.InstallerData{
		Image:          chosenImage,
		Disk:           chosenDisk,
		TypeFilesystem: chosenFilesystem,
		TypeBoot:       chosenBootMode,
		User:           user,
	}

	installService := install.NewInstallerService(installData)
	watchNewLog()
	watchStatus(installService, cancelBtn)
	//go installService.RunInstall()

	return outerBox
}

// watchStatus обновляет статус и, при достижении StatusCompleted, меняет кнопку "Отмена" на "Перезагрузка".
func watchStatus(service *install.InstallerService, cancelBtn *gtk.Button) {
	go func() {
		for range service.Status.NotifyChan() {
			currentStatus := service.Status.GetStatusText()
			glib.IdleAdd(func() {
				statusLabel.SetLabel(fmt.Sprintf("<big><b>%s</b></big>", currentStatus))
				if service.Status.GetStatus() == install.StatusCompleted {
					cancelBtn.SetLabel(lib.T("Restart"))
					cancelBtn.AddCSSClass("blue-button")
					cancelBtn.ConnectClicked(func() {
						go func() {
							exec.Command("reboot").Run()
						}()
					})
				}
			})
		}
	}()
}

// watchNewLog считывает вывод логгера и обновляет logView.
func watchNewLog() {
	re := regexp.MustCompile(`level=(\w+).*msg="(.*?)"`)
	var lastLen int
	go func() {
		for range lib.LogBuffer.NotifyChan() {
			newText := lib.GetLogText()
			glib.IdleAdd(func() bool {
				buffer := logView.Buffer()
				tt := buffer.TagTable()
				redTag := tt.Lookup("red")
				if redTag == nil {
					redTag = gtk.NewTextTag("red")
					redTag.Object.SetObjectProperty("foreground", "red")
					tt.Add(redTag)
				}
				yellowTag := tt.Lookup("yellow")
				if yellowTag == nil {
					yellowTag = gtk.NewTextTag("yellow")
					yellowTag.Object.SetObjectProperty("foreground", "orange")
					tt.Add(yellowTag)
				}
				if len(newText) > lastLen {
					additional := newText[lastLen:]
					lastLen = len(newText)
					lines := strings.Split(additional, "\n")
					for _, line := range lines {
						line = strings.TrimSpace(line)
						if line == "" {
							continue
						}
						matches := re.FindStringSubmatch(line)
						var level, msg string
						if len(matches) >= 3 {
							level = strings.ToLower(matches[1])
							msg = matches[2]
						} else {
							msg = line
						}
						var tag *gtk.TextTag
						switch level {
						case "error":
							tag = redTag
						case "warning", "debug":
							tag = yellowTag
						default:
							tag = nil
						}
						startIter := buffer.EndIter()
						buffer.Insert(startIter, msg+"\n")
						endIter := buffer.EndIter()
						if tag != nil {
							buffer.ApplyTag(tag, startIter, endIter)
						}
					}
				}
				endIter := buffer.EndIter()
				logView.ScrollToIter(endIter, 0.0, false, 0, 1)
				return false
			})
		}
	}()
}
