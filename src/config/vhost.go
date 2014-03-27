package config

import (
	"encoding/xml"
	//"path"
	"path/filepath"
	"os"
	"strings"
	"strconv"
	"fmt"
)

type Vhost struct {
	//XMLName xml.Name `xml:"vhost"`
	Bind []string `xml:"bind"`
	Host []Host   `xml:"host"`
}

type Host struct {
	Ip     string `xml:"ip,attr"`
	Port   int    `xml:"port,attr"`
	Domain string `xml:",chardata"`
}

func LoadVhostDir() {
	newsites := make(map[Sites]int)
	newlisten	:= make(map[int]int)
	newvhosts := make(map[int]Vhost)
	index := 1000	
	for _, dir := range config.VhostDir {
		files, err := filepath.Glob(configPath + dir)
		if  err != nil {
			fmt.Printf("read dir %s , %s", configPath + dir, err.Error())
			continue
		}
		for _, filename := range files {
			file, err := os.Open(filename)
			defer file.Close()
			if err != nil {
				fmt.Printf("open error: %s", err.Error())
				return
			}
			vhost := Vhost{}
			xmlObj := xml.NewDecoder(file)
			err = xmlObj.Decode(&vhost)
			if err != nil {
				fmt.Printf("vhost xml parse error: %s\n", err.Error())
			}
			newvhosts[index] = vhost
				for _, bind := range vhost.Bind {
						ip, port := getBindIpPort(bind)
						newlisten[port] ++
						for _, host := range vhost.Host {
							fmt.Printf("%s:%d%s\n", ip, port, host.Domain)
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