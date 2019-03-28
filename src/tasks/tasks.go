package tasks

import (
	"errors"
	"fmt"
	"github.com/jar3b/concron/src/helpers"
	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

func LoadTasks(configFileName string) (*ConfigDescriptiveInfo, error) {
	config := ConfigDescriptiveInfo{}
	data, err := helpers.ReadBinFile(configFileName)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal([]byte(data), &config); err != nil {
		return nil, err
	}

	// init tasks
	errList := config.InitTasks()
	for _, err := range errList {
		log.Errorf(fmt.Sprintf("parse task error: %v", err))
	}
	if len(errList) > 0 {
		return nil, errors.New("cannot parse tasks, errors was shown above")
	}

	return &config, nil
}
