package profile

import (
	"context"
	"errors"
	"io/ioutil"
	"net"
	"net/http"
	"os"

	"github.com/Dreamacro/clash/adapters/inbound"
	"github.com/Dreamacro/clash/component/socks5"
	"github.com/Dreamacro/clash/config"
	"github.com/Dreamacro/clash/constant"
	"github.com/Dreamacro/clash/tunnel"
)

const defaultFileMode = 0600

var client = &http.Client{
	Transport: &http.Transport{
		DialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			if network != "tcp" && network != "tcp4" && network != "tcp6" {
				return nil, errors.New("Unsupported network type " + network)
			}

			client, server := net.Pipe()

			tunnel.Instance().Add(inbound.NewSocket(socks5.ParseAddr(address), server, constant.HTTP, constant.TCP))

			go func() {
				if ctx == nil || ctx.Done() == nil {
					return
				}

				<-ctx.Done()

				client.Close()
				server.Close()
			}()

			return client, nil
		},
	},
}

func DownloadAndCheck(url, output, baseDir string) error {
	response, err := client.Get(url)
	if err != nil {
		return err
	}

	defer response.Body.Close()
	data, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return err
	}

	return SaveAndCheck(data, output, baseDir)
}

func ReadAndCheck(fd int, output, baseDir string) error {
	file := os.NewFile(uintptr(fd), "/dev/null")

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	return SaveAndCheck(data, output, baseDir)
}

func SaveAndCheck(data []byte, output, baseDir string) error {
	original := constant.Path.HomeDir()
	constant.SetHomeDir(baseDir)
	defer constant.SetHomeDir(original)

	_, err := parseConfig(data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(output, data, defaultFileMode)
}

func MoveAndCheck(source, target, baseDir string) error {
	original := constant.Path.HomeDir()
	constant.SetHomeDir(baseDir)
	defer constant.SetHomeDir(original)

	buf, err := ioutil.ReadFile(source)
	if err != nil {
		return err
	}

	_, err = parseConfig(buf)
	if err != nil {
		return err
	}

	if err := ioutil.WriteFile(target, buf, defaultFileMode); err != nil {
		return err
	}

	os.Remove(source)

	return nil
}

func parseConfig(data []byte) (*config.Config, error) {
	raw, err := config.UnmarshalRawConfig(data)
	if err != nil {
		return nil, err
	}

	raw.ExternalUI = ""
	raw.ExternalController = ""

	return config.ParseRawConfig(raw)
}