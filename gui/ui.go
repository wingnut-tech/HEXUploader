package main

import (
	"fmt"
	"net/url"
	"runtime"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/wingnut-tech/HEXUploader/arduino"
	"github.com/wingnut-tech/HEXUploader/common"
	"github.com/wingnut-tech/HEXUploader/state"
	"github.com/wingnut-tech/HEXUploader/update"
)

type UI struct {
	app   fyne.App
	state *state.State

	mainWindow   fyne.Window
	flashSection *fyne.Container

	portList *widget.Select

	statusBar *widget.Label
}

func createFlashSection(ui *UI) {
	ui.portList = widget.NewSelect([]string{}, func(value string) {
		ui.state.CurrentPort = value
		if value == "" {
			ui.state.Ready.PortSelected = false
		} else {
			ui.state.Ready.PortSelected = true
		}
	})
	ui.portList.PlaceHolder = "(Select COM port)"

	filenameLabel := widget.NewLabel("No File Selected")
	filenameLabel.Wrapping = fyne.TextWrapBreak

	openBtn := widget.NewButton("Browse for file", func() {
		d := ui.app.NewWindow("Open")
		d.Resize(fyne.Size{Width: 800, Height: 600})
		d.Show()

		fd := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			defer d.Close()

			if err != nil {
				dialog.ShowError(err, ui.mainWindow)
				return
			}
			if reader == nil {
				return
			}

			ui.state.HexFile = reader.URI().Path()
			filenameLabel.SetText(ui.state.HexFile)
			ui.state.Ready.HexSelected = true
		}, d)
		fd.SetFilter(storage.NewExtensionFileFilter([]string{".hex"}))

		// TODO: fix this somehow
		go func() {
			for {
				if (d.Canvas().Size() != fyne.Size{Width: 0, Height: 0}) {
					break
				}
				time.Sleep(10 * time.Millisecond)
			}
			fd.Resize(fyne.Size{Width: 6000, Height: 4000})

			fd.Show()
		}()

	})

	flashBtn := widget.NewButton("Flash Firmware", func() {
		if ui.state.CheckReady() {
			go func() {
				ui.state.Ready.NotFlashing = false
				err := arduino.FlashHex(ui.state)
				if err != nil {
					ui.state.SetStatus(err.Error())
				} else {
					ui.state.SetStatus("Done!")
				}
				ui.state.Ready.NotFlashing = true
			}()
		}
	})

	ui.flashSection = container.NewVBox(
		container.NewGridWithColumns(2,
			container.NewVBox(
				ui.portList,
				openBtn,
			),
			flashBtn,
		),
		filenameLabel,
	)
}

func createMainWindow(ui *UI) *fyne.Container {
	createFlashSection(ui)

	titleLabel := widget.NewLabel("WingnutTech HEX Uploader " + state.APP_VERSION)
	titleLabel.Alignment = fyne.TextAlignCenter

	driverBtn := widget.NewButton("CH340 Drivers", func() {
		go common.InstallCH340(ui.state)
	})
	// CH340 driver button on windows only
	if runtime.GOOS != "windows" {
		driverBtn.Hide()
	}

	ui.statusBar = widget.NewLabel("")
	ui.statusBar.Wrapping = fyne.TextTruncate

	mainSection := container.NewVBox(
		titleLabel,
		driverBtn,
		ui.flashSection,
	)

	return container.NewVBox(mainSection, ui.statusBar)
}

func updatePopup(ver string, ui *UI) {
	popup := ui.app.NewWindow("Update")
	var content *fyne.Container

	if runtime.GOOS == "darwin" {
		label := widget.NewLabel("New version available: " + ver + ".\n")
		link, _ := url.Parse(fmt.Sprintf("%s/%s/%s", update.APP_RELEASE_URL_PREFIX, "tag", ver))
		hyperlink := widget.NewHyperlink("Download", link)

		content = container.NewVBox(
			label,
			container.NewHBox(
				hyperlink,
				layout.NewSpacer(),
			),
			widget.NewButton("OK", func() {
				popup.Close()
			}),
		)
	} else {
		label := widget.NewLabel("New version available: " + ver + ".\n\nWould you like to automatically install the update?")
		content = container.NewVBox(
			label,
			layout.NewSpacer(),
			container.NewGridWithColumns(2,
				widget.NewButton("No", func() {
					popup.Close()
				}),
				widget.NewButton("Yes", func() {
					popup.Close()

					err := update.UpdateApp(ver, ui.state, func() { ui.app.Quit() })
					if err != nil {
						ui.state.SetStatus(err.Error())
					}
				}),
			),
		)
	}

	popup.SetContent(content)
	popup.CenterOnScreen()
	popup.Show()
}

func (ui *UI) setStatus(text string) {
	ui.statusBar.SetText(text)
	fmt.Println(text)
}

func (ui *UI) resizeMainWindow() {
	ui.mainWindow.Resize(ui.mainWindow.Content().MinSize())
}
