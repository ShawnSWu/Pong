package core

import (
	"fmt"
	"github.com/spf13/cast"
	"github.com/spf13/viper"
)

func ReadProperties(env string) (string, string) {
	viper.SetConfigName(fmt.Sprintf("%s/%s", "properties", env))
	viper.SetConfigType("properties")
	viper.AddConfigPath("./")

	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("Fatal error config file: %w \n", err))
	}

	host := cast.ToString(viper.Get("HOST_IP"))
	port := cast.ToString(viper.Get("HOST_PORT"))
	return host, port
}
