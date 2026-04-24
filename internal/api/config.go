package api

import "fmt"

type Config struct {
	Listen                string
	LegacyListen          string
	Debug                 bool
	AppVersion            string
	AllowedOrigins        []string
	ContentSecurityPolicy string
}

func (c Config) Validate() error {
	_, _, err := newCrossOriginProtection(c.AllowedOrigins)
	if err != nil {
		return fmt.Errorf("invalid API allowed origin: %w", err)
	}

	return nil
}
