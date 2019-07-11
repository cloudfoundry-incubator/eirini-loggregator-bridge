package podwatcher

import (
	config "github.com/SUSE/eirini-loggregator-bridge/config"
)

type PodWatcher struct {
	Config *config.ConfigType
}

func NewPodWatcher(config *config.ConfigType) *PodWatcher {
	return &PodWatcher{Config: config}
}
