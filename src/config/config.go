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
	RedisServer     RedisServer  `xml:"redis_server"`
}

type RedisServer struct {
	PoolSize     int `xml:"pool_size,attr"`
	Password   string    `xml:"password,attr"`
	Addr string `xml:",chardata"`	
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
	listen map[string]int
	ssl_listen map[string][]Ssl
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
		logger.Crash(err.Error())
		return
	}
	xmlObj := xml.NewDecoder(file)
	err = xmlObj.Decode(&config)

	if err != nil {
		logger.Crash(err)
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

func GetHostname() string {
	return config.Hostname
}

func GetRedis() RedisServer {
	return config.RedisServer
}