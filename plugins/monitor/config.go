package main

import "time"

type Config struct {
	Enable      bool          `yaml:"enable"`
	Interval    time.Duration `yaml:"interval"`
	LarkWebHook string        `yaml:"larkWebHook"`
}
