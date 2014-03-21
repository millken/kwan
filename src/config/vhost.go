package config

import (
	"encoding/xml"
	//"path"
	"path/filepath"
	"os"
	"strings"
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
				for _, bind := range vhost.Bind {
					val, ok := configBind[bind]
					if ok {
						for _, host := range vhost.Host {
							if strings.HasPrefix(host.Domain, "*") == false {
								if stringInSlice(host.Domain, val.WideDomain) {
									break
								}else{
									val.WideDomain = append(val.WideDomain, host.Domain)
								}

							}else{
								if stringInSlice(host.Domain, val.Domain) {
									break
								}else{
									val.Domain = append(val.Domain, host.Domain)
								}
							}
						}
						
					}else{
						val = ConfigBind{}
						for _, host := range vhost.Host {
							if strings.HasPrefix(host.Domain, "*") == false {
								if stringInSlice(host.Domain, val.WideDomain) {
									break
								}else{
									val.WideDomain = append(val.WideDomain, host.Domain)
								}

							}else{
								if stringInSlice(host.Domain, val.Domain) {
									break
								}else{
									val.Domain = append(val.Domain, host.Domain)
								}
							}
						}
					}
					configBind[bind] = val
				}
			config.Vhosts = append(config.Vhosts, vhost)
		}
	}
	
}

func stringInSlice(a string, list []string) bool {
    for _, b := range list {
        if b == a {
            return true
        }
    }
    return false
}