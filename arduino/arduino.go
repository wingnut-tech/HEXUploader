package arduino

import (
	"bytes"
	"context"
	"io"
	"path/filepath"
	"time"

	"github.com/arduino/arduino-cli/commands/board"
	"github.com/arduino/arduino-cli/commands/core"
	"github.com/arduino/arduino-cli/commands/upload"
	rpc "github.com/arduino/arduino-cli/rpc/cc/arduino/cli/commands/v1"
	"github.com/wingnut-tech/HEXUploader/state"
	"go.bug.st/serial"
)

const (
	FQBN    = "arduino:avr:nano:cpu=atmega328"
	FQBNold = "arduino:avr:nano:cpu=atmega328old"
)

var neededCores = [][]string{
	{"arduino", "avr"},
}

func CheckCores(s *state.State) error {
	for _, cores := range neededCores {
		if _, err := core.PlatformInstall(context.Background(), &rpc.PlatformInstallRequest{
			Instance:        s.Instance,
			PlatformPackage: cores[0],
			Architecture:    cores[1],
		}, func(curr *rpc.DownloadProgress) { s.SetStatus(curr.String()) }, func(msg *rpc.TaskProgress) { s.SetStatus(msg.String()) }); err != nil {
			return err
		}
	}
	return nil
}

func WatchPorts(s *state.State, callback func()) {
	eventsChan, _, err := board.Watch(&rpc.BoardListWatchRequest{Instance: s.Instance})
	if err != nil {
		s.SetStatus(err.Error())
	}

	// loop forever listening for board.Watch to give us events
	for event := range eventsChan {
		port := event.Port.Port
		addr := port.Address
		if event.EventType == "add" {
			s.Ports[addr] = port
			s.CurrentPort = addr
		} else {
			// if the port was in the list, remove it
			delete(s.Ports, addr)
			// if the unplugged board was our current one, grab the first available (if there is one)
			if s.CurrentPort == addr {
				if len(s.Ports) > 0 {
					for p := range s.Ports {
						s.CurrentPort = p
						break
					}
				} else {
					s.CurrentPort = ""
				}
			}
		}
		if callback != nil {
			callback()
		}
	}
}

func FlashHex(s *state.State) error {
	port := s.Ports[s.CurrentPort]

	bl := FQBNold

	tb, err := testBootloaderType(port.Address, 115200)
	if err != nil {
		return err
	}
	if tb {
		bl = FQBN
	} else {
		_, err := testBootloaderType(port.Address, 57600)
		if err != nil {
			return err
		}
	}

	if err := upload.Upload(context.Background(), &rpc.UploadRequest{
		Instance:   s.Instance,
		Fqbn:       bl,
		SketchPath: filepath.Dir(s.HexFile),
		Port:       port,
		ImportFile: s.HexFile,
	}, io.Discard, io.Discard); err != nil {
		return err
	}

	return nil
}

func testBootloaderType(p string, b int) (bool, error) {
	syncCmd := []byte{0x30, 0x20}
	inSyncResp := []byte{0x14, 0x10}
	delay := (250 * time.Millisecond)
	shortDelay := (50 * time.Millisecond)
	timeout := (250 * time.Millisecond)

	port, err := serial.Open(p, &serial.Mode{BaudRate: b})
	if err != nil {
		return false, err
	}
	defer port.Close()

	port.SetReadTimeout(timeout)

	// reset bootloader
	port.SetDTR(false)
	port.SetRTS(false)
	time.Sleep(delay)

	port.SetDTR(true)
	port.SetRTS(true)
	time.Sleep(shortDelay)

	port.ResetInputBuffer()

	for i := 0; i < 4; i++ {
		port.Write(syncCmd)
		time.Sleep(shortDelay)

		resp := make([]byte, 2)
		port.Read(resp)
		if bytes.Equal(resp, inSyncResp) {
			return true, nil
		}
		port.ResetInputBuffer()
	}
	return false, nil
}
