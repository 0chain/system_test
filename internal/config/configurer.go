package config

import (
	"github.com/0chain/system_test/internal/model"
	"github.com/go-playground/validator/v10"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"os"
)

var (
	errEmptyConfig      = errors.New("failed to read config file")
	defaultFileLocation = "/etc/system_test.yaml"
)

type Configurer struct {
	cfg            *viper.Viper
	RequiredConfig *RequiredConfig
}

func ReadConfig(fileLocation string) (*Configurer, error) {
	v := viper.New()
	if fileLocation == "" {
		fileLocation = defaultFileLocation
	}
	v.SetConfigFile(fileLocation)
	err := v.ReadInConfig()
	if err != nil {
		log.Errorf("failed to read configuration file %v", err)
		return nil, errors.Wrap(err, "failed to read configuration file")
	}

	requiredConfig := &RequiredConfig{
		DNSHostName: nil,
		LogLevel:    nil,
	}
	err = v.Unmarshal(requiredConfig)
	if err != nil {
		log.Errorf("failed to unmarshal configuration %v", err)
		return nil, errors.Wrap(err, "failed to unmarshal configuration")
	}

	validate := validator.New()

	if err := validate.Struct(requiredConfig); err != nil {
		return nil, errors.Wrap(err, "missing configuration values")
	}

	return &Configurer{
		cfg:            v,
		RequiredConfig: requiredConfig,
	}, nil
}

func WriteConfig(fileLocation string, config model.Config) error {
	data, err := yaml.Marshal(&config)

	if err != nil {
		return err
	}

	if _, err := os.Stat("./temp"); os.IsNotExist(err) {
		err = os.Mkdir("./temp", 0755)

		if err != nil {
			return err
		}
	}

	return ioutil.WriteFile("./temp/"+fileLocation, data, 0644)
}
