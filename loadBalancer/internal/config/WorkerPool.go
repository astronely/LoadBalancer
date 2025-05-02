package config

import (
	"errors"
	"gopkg.in/yaml.v3"
	"log/slog"
	"os"
)

var _ WorkerPoolConfig = (*workerPoolConfig)(nil)

type workerPoolConfig struct {
	WPoolSize  int `yaml:"pool_size"`
	WQueueSize int `yaml:"queue_size"`
}

func NewWorkerPoolConfig() (WorkerPoolConfig, error) {
	configFile, err := os.Open(configName)
	if err != nil {
		return nil, errors.New("no config file found with name " + configName)
	}
	defer configFile.Close()

	var cfg struct {
		WorkerPoolConfig workerPoolConfig `yaml:"workerPool"`
	}

	d := yaml.NewDecoder(configFile)

	if err = d.Decode(&cfg); err != nil {
		return nil, errors.New("error parsing config file " + configName)
	}

	slog.Debug("Config",
		"cfg", cfg)

	return &cfg.WorkerPoolConfig, nil
}

func (w *workerPoolConfig) PoolSize() int {
	return w.WPoolSize
}

func (w *workerPoolConfig) QueueSize() int {
	return w.WQueueSize
}
