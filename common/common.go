package common

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/wingnut-tech/HEXUploader/arduino"
	"github.com/wingnut-tech/HEXUploader/state"
	"github.com/wingnut-tech/HEXUploader/utils"
)

const (
	CH340_URL = "https://github.com/wingnut-tech/HEXUploader/releases/download/v1.0.0/CH34x_Install_Windows_v3_4.zip"
	CH340_EXE = "CH34x_Install_Windows_v3_4.EXE"
)

func InstallCH340(s *state.State) {
	// ch340 drivers are only needed on windows, because windows is so special.
	if runtime.GOOS != "windows" {
		return
	}

	s.SetStatus("Downloading CH340 drivers")

	zipFile := filepath.Join(s.TmpDir, "ch340.zip")
	exe := filepath.Join(s.TmpDir, CH340_EXE)

	if _, err := os.Stat(exe); err != nil {
		if _, err := os.Stat(zipFile); err != nil {
			err := utils.DownloadFile(zipFile, CH340_URL)
			if err != nil {
				s.SetStatus(err.Error())
				return
			}
		}

		_, err = utils.UnzipFile(zipFile, s.TmpDir)
		if err != nil {
			s.SetStatus(err.Error())
			return
		}
	}

	time.Sleep(time.Second * 2)

	cmd := exec.Command(exe)
	err := cmd.Start()
	if err != nil {
		s.SetStatus(err.Error())
		return
	}
	s.SetStatus("Started CH340 installer")
}

func Init(s *state.State) {
	s.SetStatus("Checking arduino core...")
	err := arduino.CheckCores(s)
	if err != nil {
		s.SetStatus("Error: " + err.Error())
	} else {
		s.Ready.CoreInstalled = true
	}

	s.SetStatus("Ready")
}
