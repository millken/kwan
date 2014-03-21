package config

import (
	"encoding/xml"
	"flag"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"time"
)

type Config struct {
	XMLName  xml.Name      `xml:"config"`
	Timeout  time.Duration `xml:"timeout"`
	Vhosts	[]Vhost
	VhostDir []string      `xml:"vhost_dir"`
}

type ConfigBind struct {
	WideDomain []string
	Domain []string
}

var config Config
var configFile string
var configPath string
var configBind map[string]ConfigBind

func init() {
	flag.StringVar(&configFile, "c", "config.xml", "config file path")
	flag.Parse()
	if configFile == "" {
		configFile = "../etc/config.xml"
	}
	configBind = make(map[string]ConfigBind)
}

func Read() {

	//config = Config{} //清空config
	file, err := os.Open(configFile)
	defer file.Close()
	if err != nil {
		fmt.Printf("open error: %s", err.Error())
		return
	}
	xmlObj := xml.NewDecoder(file)
	err = xmlObj.Decode(&config)

	if err != nil {
		fmt.Printf("error: %v", err)
	}
	fmt.Println("Timeout", config.Timeout)
	fmt.Printf("VhostDir:%v\n", config.VhostDir)
	current_file, _ := filepath.Abs(configFile)
	configPath = path.Dir(current_file) + "/"
	fmt.Printf("configPath:%s\n", configPath)

	LoadVhostDir()
	fmt.Printf("config:%v\n", config)
	fmt.Printf("configBind:%v\n", configBind)
}
