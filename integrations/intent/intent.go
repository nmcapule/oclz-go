package intent

import (
	"github.com/nmcapule/oclz-go/integrations/models"
)

const Vendor = "DEFAULT"

// TODO(nmcapule): If there are configs for intent tenant, add here.
type Config struct {
}

type Client struct {
	*models.BaseDatabaseTenant
	Config *Config
}
