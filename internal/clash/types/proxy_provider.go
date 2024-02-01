package types

type ProxyProvider struct {
	Type        string      `yaml:"type"`
	Path        string      `yaml:"path"`
	URL         string      `yaml:"url,omitempty"`
	Interval    int         `yaml:"interval,omitempty"`
	Filter      string      `yaml:"filter,omitempty"`
	HealthCheck HealthCheck `yaml:"health-check,omitempty"`
}

type HealthCheck struct {
	Enable   bool   `yaml:"enable"`
	URL      string `yaml:"url"`
	Interval int    `yaml:"interval"`
	Lazy     bool   `yaml:"lazy,omitempty"`
}
