package lidarr

import (
	"github.com/poiley/nebularr-operator/internal/adapters"
)

func init() {
	adapters.Register(&Adapter{})
}
