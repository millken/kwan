package main

import (
	"github.com/stvp/go-toml-config"
	"fmt"
)

var (
	Configfile = "./etc/config.conf"
  country            = config.String("country", "Unknown")
  atlantaEnabled     = config.Bool("atlanta.enabled", false)
  alantaPopulation   = config.Int("atlanta.population", 0)
  atlantaTemperature = config.Float64("atlanta.temperature", 0)
)


func SetConfigFile(input string) {
	fmt.Printf("Config filename set to %s", input)
	Configfile = input
}

func AutoReload() {
}

func ReadConfig() {
	
	fmt.Printf("Reading %s\n", Configfile)
	if err := config.Parse(Configfile); err != nil {
	panic(err)
	}
	fmt.Printf("country= %s \n", *country)
}

