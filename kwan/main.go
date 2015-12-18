package main

import (
	"fmt"
	"os"

	"github.com/jawher/mow.cli"
	"github.com/millken/kwan/service"
)

func main() {

	kwan := cli.App("kwan", "CDN server backed by etcd")
	kwan.Spec = "[-c]"
	configfile := kwan.StringOpt("c config", "/etc/kwan.conf", "config file")
	kwan.Command("run", "Run cdn server", func(cmd *cli.Cmd) {
		cmd.Action = func() {
			fmt.Printf("configfile= %s\n", *configfile)
			master, _, err := service.LoadConfig(*configfile)
			if err != nil {
				fmt.Printf("load config : %s", err)
			} else if err := service.Run(master); err != nil {
				fmt.Println(err)
			}
		}
	})

	kwan.Run(os.Args)
}
