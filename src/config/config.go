package config

import (
	//"github.com/valyala/ybc/bindings/go/ybc"
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


//map[ip][port][domain][rule_id]
//type Sites map[string]map[int]map[string]int

//http://blog.golang.org/go-maps-in-action
type Sites struct {
	BindIp string
	BindPort int
	Domain string
}

var config Config
var configFile string
var configPath string
var (
	sites map[Sites]int
	listen map[int]int
	vhosts map[int]Vhost
)
func init() {
	flag.StringVar(&configFile, "c", "config.xml", "config file path")
	flag.Parse()
	if configFile == "" {
		configFile = "../etc/config.xml"
	}
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
	fmt.Printf("sites:%v\n", sites)
}


func GetListen() map[int]int {
	return listen
}