package main

import (
	"github.com/realHoangHai/golb/cmd/server"
	"github.com/realHoangHai/golb/internal/config"
	"github.com/spf13/viper"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func load(path string) (cfg config.Config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName("config-weighted")
	viper.SetConfigType("yaml")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&cfg)
	return
}

func main() {
	cfg, err := load("./config")
	if err != nil {
		log.Fatal("cannot load config:", err)
	}

	sigs := make(chan os.Signal, 1)

	server.NewServer(&cfg).Run()

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	<-sigs
	log.Println("Exiting")
}
