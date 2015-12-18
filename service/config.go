package service

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type MasterConfig struct {
	Maxprocs       int      `toml:"maxprocs"`
	EtcdNodes      []string `toml:"etcd_nodes"`
	PoolSize       int      `toml:"poolsize"`
	CpuProfName    string   `toml:"cpuprof"`
	MemProfName    string   `toml:"memprof"`
	BaseDir        string   `toml:"base_dir"`
	PidFile        string   `toml:"pid_file"`
	Hostname       string
	MaxMessageSize uint32 `toml:"max_message_size"`
}

func ReplaceEnvsFile(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	contents, err := ioutil.ReadAll(file)
	if err != nil {
		return "", err
	}
	return string(contents), nil
}

func LoadConfig(configPath string) (masterConfig *MasterConfig, plugConfig map[string]toml.Primitive, err error) {
	hostname, err := os.Hostname()
	if err != nil {
		return
	}

	masterConfig = &MasterConfig{Maxprocs: 1,
		PoolSize:    100,
		EtcdNodes:   nil,
		CpuProfName: "",
		MemProfName: "",
		PidFile:     "",
		Hostname:    hostname,
	}

	var configFile map[string]toml.Primitive
	p, err := os.Open(configPath)
	if err != nil {
		return nil, nil, fmt.Errorf("Error opening config file: %s", err)
	}
	fi, err := p.Stat()
	if err != nil {
		return nil, nil, fmt.Errorf("Error fetching config file info: %s", err)
	}

	if fi.IsDir() {
		files, _ := ioutil.ReadDir(configPath)
		for _, f := range files {
			fName := f.Name()
			if !strings.HasSuffix(fName, ".toml") {
				// Skip non *.toml files in a config dir.
				continue
			}
			fPath := filepath.Join(configPath, fName)
			contents, err := ReplaceEnvsFile(fPath)
			if err != nil {
				return nil, nil, err
			}
			if _, err = toml.Decode(contents, &configFile); err != nil {
				return nil, nil, fmt.Errorf("Error decoding config file: %s", err)
			}
		}
	} else {
		contents, err := ReplaceEnvsFile(configPath)
		if err != nil {
			return nil, nil, err
		}
		if _, err = toml.Decode(contents, &configFile); err != nil {
			return nil, nil, fmt.Errorf("Error decoding config file: %s", err)
		}
	}

	parsed_config, ok := configFile["master"]
	if ok {
		if err = toml.PrimitiveDecode(parsed_config, masterConfig); err != nil {
			err = fmt.Errorf("Can't unmarshal master config: %s", err)
		}
	}
	plugConfig = configFile
	delete(plugConfig, "master")
	return
}
