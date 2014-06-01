package config

import (
	"encoding/xml"
	"flag"
	"logger"
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
	Hostname  string  `xml:"hostname"`
	Servename  string  `xml:"servername"`
	Console string  `xml:"console"`
	CacheSetting     CacheSetting  `xml:"cache_setting"`
}

type CacheSetting struct {
	Path     string `xml:"path,attr"`
	HotItem   int    `xml:"hot_item,attr"`	
}
//map[ip][port][domain][rule_id]
//type Sites map[string]map[int]map[string]int

//http://blog.golang.org/go-maps-in-action
type Sites struct {
	BindIp string
	BindPort int
	Domain string
}

var config *Config
var configFile string
var debugMode  int
var configPath string
var (
	sites map[Sites]int
	listen map[string]int
	ssl_listen map[string][]Ssl
	vhosts map[int]Vhost
	ups map[string]Ups
)
func init() {
	flag.StringVar(&configFile, "c", "", "config file path")
	flag.IntVar(&debugMode, "d", 4, "debug level. 0=FINEST,1=FINE,2=DEBUG,3=TRACE,4=INFO(default),5=WARNING,6=ERROR,7=CRITICAL")
	flag.Parse()
	if configFile == "" {
		configFile = "/etc/kwan/config.xml"
	}
	if os.Geteuid() != 0 {
		logger.Error("please run as root")
	}
	logger.Global = logger.NewDefaultLogger(logger.Level(debugMode))
}

func Read()  {
	logger.Info("load main config: %s", configFile)
	//config = Config{} //清空config
	file, err := os.Open(configFile)
	if err != nil {
		logger.Exit(err.Error())
	}else{
		defer file.Close()
	}
	xmlObj := xml.NewDecoder(file)
	err = xmlObj.Decode(&config)

	if err != nil {
		logger.Exit(err.Error())
	}
	current_file, _ := filepath.Abs(configFile)
	configPath = path.Dir(current_file) + "/"

	LoadVhostDir()
	//fmt.Printf("config:%v\n", config)
	//fmt.Printf("sites:%v\n", sites)
}


func GetListen() map[string]int {
	return listen
}

func GetSslListen() map[string][]Ssl {
	return ssl_listen
}

func GetUpstream() map[string]Ups {
	return ups
}

func GetHostname() string {
	return config.Hostname
}

func GetServername() string {
	return config.Servename
}
func GetConsoleAddr() string {
	return config.Console
}

func GetCacheSetting() CacheSetting {
	if config.CacheSetting.Path == "" {
		config.CacheSetting.Path = os.TempDir()
	}
	if config.CacheSetting.HotItem == 0 {
		config.CacheSetting.HotItem = 10000
	}	
	return config.CacheSetting
}