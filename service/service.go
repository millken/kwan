package service

import (
	"fmt"
	"os"

	etcd "github.com/coreos/etcd/client"
	log "github.com/millken/log4go"
)

type Service struct {
	client etcd.Client
	master *MasterConfig
	sigC   chan os.Signal
}

func Run(master *MasterConfig) error {
	service := NewService(master)
	if err := service.Start(); err != nil {
		log.Error("Failed to start service: %v", err)
		return fmt.Errorf("service start failure: %s", err)
	} else {
		log.Info("Service exited gracefully")
	}
	return nil
}
func NewService(master *MasterConfig) *Service {
	log.Register(log.NewConsoleWriter())
	log.SetLevel(log.DEBUG)
	return &Service{master: master, sigC: make(chan os.Signal, 1024)}
}

func (s *Service) Start() error {
	log.Info("procs=%d\netcd=%#v", s.master.Maxprocs, s.master.EtcdNodes)
	for {
		select {
		case signal := <-s.sigC:
			switch signal {
			}
		}
	}
	return fmt.Errorf("xxx")
}
