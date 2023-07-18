package state

import (
	"os"
	"path/filepath"

	"github.com/arduino/arduino-cli/commands"
	"github.com/arduino/arduino-cli/configuration"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	"github.com/sirupsen/logrus"
)

const (
	APP_NAME     = "HEX Uploader"
	APP_VERSION  = "v1.0.0"
	TMP_DIR_NAME = "HEXUploader"
)

type State struct {
	Instance *rpc.Instance
	Ready    Ready
	TmpDir   string

	HexFile string

	Ports       map[string]*rpc.Port
	CurrentPort string

	StatusFunc func(text string)

	AppType string
}

type Ready struct {
	PortSelected  bool
	NotFlashing   bool
	CoreInstalled bool
	HexSelected   bool
}

func NewState(appType string, statusFunc func(text string)) (*State, error) {
	s := &State{}

	s.AppType = appType
	s.StatusFunc = statusFunc

	// arduino-cli config
	configuration.Settings = configuration.Init("")
	logrus.SetLevel(logrus.FatalLevel)

	res, err := commands.Create(&rpc.CreateRequest{})
	if err != nil {
		return nil, err
	}

	s.Instance = res.Instance
	commands.Init(&rpc.InitRequest{Instance: s.Instance}, func(res *rpc.InitResponse) {
		if st := res.GetError(); st != nil {
			s.SetStatus("Error initializing instance: " + st.Message)
		}
	})

	tmpDir := os.TempDir()
	if tmpDir != "" {
		tmpDirSym, err := filepath.EvalSymlinks(tmpDir)
		if err != nil {
			return nil, err
		}

		tmpDir = filepath.Join(tmpDirSym, TMP_DIR_NAME)
		os.MkdirAll(tmpDir, 0777)
		os.Chdir(tmpDir)
	}
	s.TmpDir = tmpDir

	s.Ports = make(map[string]*rpc.Port)

	s.Ready = Ready{
		NotFlashing: true,
	}

	return s, nil
}

func (s *State) SetStatus(text string) {
	if s.StatusFunc != nil {
		s.StatusFunc(text)
	}
}

func (s *State) CheckReady() bool {
	switch {
	case !s.Ready.PortSelected:
		s.SetStatus("No port selected")
	case !s.Ready.CoreInstalled:
		s.SetStatus("Arduino core still installing")
	case !s.Ready.HexSelected:
		s.SetStatus("No file selected")
	case !s.Ready.NotFlashing:
	default:
		return true
	}
	return false
}
