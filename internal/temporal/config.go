package temporal

import "github.com/spf13/viper"

type Config struct {
	Address   string
	Namespace string
	TaskQueue string
}

func SetDefaults(v *viper.Viper) {
	v.SetDefault("temporal.address", "127.0.0.1:7233")
	v.SetDefault("temporal.namespace", "default")
	v.SetDefault("temporal.taskQueue", "global")
}
