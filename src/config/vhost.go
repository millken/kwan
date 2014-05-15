package config

import (
	"encoding/xml"
	//"path"
	"path/filepath"
	"os"
	"strings"
	"strconv"
	"fmt"
	"logger"
)

type Vhost struct {
	Name string `xml:"name,attr"`
	Status int `xml:"status,attr"`
	Bind []string `xml:"bind"`
	Host []Host   `xml:"host"`
	Cache []Cache `xml:"cache"`
	Limit Limit `xml:"limit"`
	Ssl []Ssl `xml:"ssl"`
	Ddos Ddos `xml:"ddos"`
	WhiteList BlackWhiteList `xml:"whitelist"`
	BlackList BlackWhiteList `xml:"blacklist"`
	Log Log `xml:"log"`
	Request Request `xml:"request"`
}

type Request struct {
	Http2https []string `xml:"http2https"`
}

type Cache struct {
	Base bool `xml:"base,attr"`
	Time int32 `xml:"time,attr"`
	FileExt string `xml:"file_ext,attr"`
	Static bool `xml:"static,attr"`
 	Nocache bool `xml:"nocache,attr"`
 	Url string `xml:"url,attr"`
 	Regex string `xml:"regex,attr"`
 	IgnoreParam bool `xml:"ignore_param,attr"`
}

type Ssl struct {
	Bind string `xml:"bind,attr"`
	Sort int `xml:"sort,attr"`
	Keyfile string `xml:"key_file,attr"`
	Certfile string `xml:"cert_file,attr"`
}

type Limit struct {
	Timeout int `xml:"timeout,attr"`
	Speed int `xml:"speed,attr"`
}

type Host struct {
	Ip     string `xml:"ip,attr"`
	Port   int    `xml:"port,attr"`
	Domain string `xml:",chardata"`
}

type Ddos struct {
	Request int64 `xml:"request,attr"`
	Rtime int32 `xml:"rtime,attr"`
	Stime int32 `xml:"stime,attr"`	
	Mode int32 `xml:"mode,attr"`
}

type BlackWhiteList struct {
	Ip  []string `xml:"ip"`
	Url  []string `xml:"url"`
	UserAgent  []string `xml:"useragent"`
}

type Log struct {
	Status bool  `xml:"status,attr"`
	Type   string `xml:"type,attr"`
	RotateDaily bool `xml:"rotate_daily,attr"`
	Addr  string `xml:",chardata"`
}
func LoadVhostDir()  {
	newsites := make(map[Sites]int)
	newlisten	:= make(map[string]int)
	newssl	:= make(map[string][]Ssl)
	newvhosts := make(map[int]Vhost)
	index := 1000	
	for _, dir := range config.VhostDir {
		files, err := filepath.Glob(configPath + dir)
		if  err != nil {
			logger.Warn("read dir %s , %s", configPath + dir, err.Error())
			continue
		}
		for _, filename := range files {
			file, err := os.Open(filename)
			defer file.Close()
			if err != nil {
				logger.Warn("open error: %s", err.Error())
				continue
			}
			vhost := Vhost{}
			xmlObj := xml.NewDecoder(file)
			err = xmlObj.Decode(&vhost)
			if err != nil {
				logger.Warn("vhost xml parse error: %s\n", err.Error())
				continue
			}
			newvhosts[index] = vhost
				for _, bind := range vhost.Bind {
						ip, port := getBindIpPort(bind)
						newlisten[fmt.Sprintf("%s:%v", ip, port)] ++
						for _, host := range vhost.Host {
							//fmt.Printf("%s:%d%s\n", ip, port, host.Domain)
							newsites[Sites{ip, port, host.Domain}] = index
						}
						
				}
				for _, ssl := range vhost.Ssl {
					ip, port := getBindIpPort(ssl.Bind)
					iport := fmt.Sprintf("%s:%v", ip, port)
					newssl[iport] = append(newssl[iport], ssl)
					for _, host := range vhost.Host {
						newsites[Sites{ip, port, host.Domain}] = index
					}					
				}
			config.Vhosts = append(config.Vhosts, vhost)
			index++ 
		}
	}
	sites = newsites
	listen = newlisten
	vhosts = newvhosts
	ssl_listen = newssl
}



func getBindIpPort(bind string) (ip string, port int) {
	ip = "0.0.0.0"
	if strings.Index(bind, ":") == -1 {
		port1, err := strconv.Atoi(bind)
		if err != nil {
			port = 80
		}else{
			port = port1
		}
	}else{
	
		tmp := strings.Split(bind, ":")
		
		if tmp[0] != "" && tmp[0] != "*" {
			ip = tmp[0]
			
			port2, err := strconv.Atoi(tmp[1])
			if err != nil {
				port = 80
			}else{
				port = port2
			}
		}else{
			port2, err := strconv.Atoi(tmp[1])
			if err != nil {
				port = 80
			}else{
				port = port2
			}			
		}

	}
	return
}
func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}