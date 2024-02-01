package clash

import (
	"github.com/ryan961/clashset/internal/clash/types"
	"gopkg.in/yaml.v3"
)

type RawConfig struct {
	Port               int              `yaml:"port"`
	SocksPort          int              `yaml:"socks-port"`
	RedirPort          int              `yaml:"redir-port"`
	TProxyPort         int              `yaml:"tproxy-port"`
	MixedPort          int              `yaml:"mixed-port"`
	Authentication     []string         `yaml:"authentication"`
	AllowLan           bool             `yaml:"allow-lan"`
	BindAddress        string           `yaml:"bind-address"`
	Mode               types.TunnelMode `yaml:"mode"`
	LogLevel           types.LogLevel   `yaml:"log-level"`
	IPv6               bool             `yaml:"ipv6"`
	ExternalController string           `yaml:"external-controller"`
	ExternalUI         string           `yaml:"external-ui"`
	Secret             string           `yaml:"secret"`
	Interface          string           `yaml:"interface-name"`
	RoutingMark        int              `yaml:"routing-mark"`
	Tunnels            []types.Tunnel   `yaml:"tunnels"`

	ProxyProvider map[string]map[string]any `yaml:"proxy-providers"`
	Hosts         map[string]string         `yaml:"hosts"`
	Inbounds      []types.Inbound           `yaml:"inbounds"`
	DNS           types.RawDNS              `yaml:"dns"`
	Profile       types.Profile             `yaml:"profile"`
	Proxy         []map[string]any          `yaml:"proxies"`
	ProxyGroup    []map[string]any          `yaml:"proxy-groups"`
	Rule          []string                  `yaml:"rules"`
}

func defaultConfig() *RawConfig {
	return &RawConfig{
		AllowLan:       false,
		BindAddress:    "*",
		Mode:           types.Rule,
		Authentication: []string{},
		LogLevel:       types.INFO,
		Hosts:          map[string]string{},
		Rule:           []string{},
		Proxy:          []map[string]any{},
		ProxyGroup:     []map[string]any{},
		DNS: types.RawDNS{
			Enable:      false,
			UseHosts:    true,
			FakeIPRange: "198.18.0.1/16",
			FallbackFilter: types.RawFallbackFilter{
				GeoIP:     true,
				GeoIPCode: "CN",
				IPCIDR:    []string{},
			},
			DefaultNameserver: []string{
				"114.114.114.114",
				"8.8.8.8",
			},
		},
		Profile: types.Profile{
			StoreSelected: true,
		},
	}
}

func Parse(buf []byte) (*RawConfig, error) {
	// config with default value
	rawCfg := defaultConfig()
	if err := yaml.Unmarshal(buf, rawCfg); err != nil {
		return nil, err
	}

	return rawCfg, nil
}
