package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"gopkg.in/yaml.v3"
)

const (
	templateFile      = "template.yaml"
	configurationFile = "configuration.yaml"

	timeout = 30 * time.Second
)

var (
	fileMode os.FileMode = 0o666
	dirMode  os.FileMode = 0o755

	httpClient = resty.New()

	urls, dir, file, out, proxy string
	interval                    uint64

	urlList     []string
	configMap   = make(map[string]any)
	proxyConfig ProxyConfig

	defaultHealthCheck = map[string]any{
		"enable":   true,
		"url":      "http://www.gstatic.com/generate_204",
		"interval": 3600,
	}
)

type ProxyConfig struct {
	Filters []Filters `yaml:"filters"`
	Groups  []struct {
		Group      Group    `yaml:"group"`
		UseFilters []string `yaml:"use_filters"`
		RmAirports bool     `yaml:"rm_airports"`
	} `yaml:"groups"`
}

type Filters struct {
	Name   string `yaml:"name"`
	Filter string `yaml:"filter"`

	provider []string `yaml:"-"`
}

type Group struct {
	Name       string   `yaml:"name" mapstructure:"name"`
	Type       string   `yaml:"type" mapstructure:"type"`
	Proxies    []string `yaml:"proxies" mapstructure:"proxies,omitempty"`
	Use        []string `yaml:"use" mapstructure:"use,omitempty"`
	URL        string   `yaml:"url" mapstructure:"url,omitempty"`
	Interval   int      `yaml:"interval" mapstructure:"interval,omitempty"`
	Lazy       bool     `yaml:"lazy" mapstructure:"lazy,omitempty"`
	DisableUDP bool     `yaml:"disable-udp" mapstructure:"disableUDP,omitempty"`
	Filter     string   `yaml:"filter" mapstructure:"filter,omitempty"`
}

type ProxySchema struct {
	Proxies []map[string]any `yaml:"proxies"`
}

func init() {
	flag.StringVar(&urls, "urls", "", "订阅链接地址 🔗 (多个订阅链接之间 \",\" 分隔)")
	flag.StringVar(&proxy, "proxy", "", "http proxy ✈️")
	flag.StringVar(&dir, "dir", "./", "配置文件地址 📁 , 默认配置文件 template.yaml & proxy-filters.yaml 文件夹")
	flag.StringVar(&file, "file", "config.yaml", "导出配置文件名称 📃")
	flag.StringVar(&out, "out", "out/", "导出文件夹 📁")
	flag.Uint64Var(&interval, "interval", 86400, "proxy-providers 配置刷新间隔时长(s)")
	flag.Parse()

	if strings.TrimSpace(urls) == "" {
		log.Fatal("urls is empty")
	}
	urlList = strings.Split(urls, ",")
	log.Println("subscribed links:", urlList)

	if proxy != "" {
		log.Println("http proxy:", proxy)
		httpClient.SetProxy(proxy)
	}
}

func main() {
	// loading template.yaml
	if err := unmarshal(filepath.Join(dir, templateFile), &configMap); err != nil {
		log.Fatal(err)
		return
	}

	// loading proxy-filters.yaml
	if err := unmarshal(filepath.Join(dir, configurationFile), &proxyConfig); err != nil {
		log.Fatal(err)
		return
	}

	airportFiles := make(map[string]map[string]any)
	// download proxy file && add proxy providers
	if _, ok := configMap["proxy-providers"]; !ok {
		configMap["proxy-providers"] = make(map[string]any)
	}
	for _, v := range urlList {
		filename, proxySchema, err := downloadProxies(v)
		if err != nil {
			log.Fatal(err)
			return
		}

		// filename 配置文件默认同级文件夹

		provider := map[string]any{
			"type":         "http",
			"path":         filename,
			"url":          v,
			"interval":     interval,
			"health-check": defaultHealthCheck,
		}

		name := path.Base(filename[:len(filename)-len(path.Ext(filename))])
		name = fmt.Sprintf(`✈️ %s`, name)

		airportFiles[name] = map[string]any{
			"path":        filename,
			"proxySchema": proxySchema,
		}

		configMap["proxy-providers"].(map[string]any)[name] = provider
	}

	proxyFilterMap := make(map[string]Filters)

	// add filter proxy providers
	for _, v := range proxyConfig.Filters {
		filterReg, err := regexp.Compile(v.Filter)
		if err != nil {
			log.Fatalf("invalid filter regex: %w", err)
			return
		}

		for name, m := range airportFiles {
			// NOTICE: 不存在 filter 条件的节点过滤不添加
			proxySchema := m["proxySchema"].(*ProxySchema)
			exists := lo.ContainsBy(proxySchema.Proxies, func(item map[string]any) bool {
				return filterReg.MatchString(item["name"].(string))
			})

			// 存在匹配到规则的 proxy
			if exists {
				// providerName: jp-sub.xsus.me
				providerName := fmt.Sprintf("%s-%s", v.Name, name)
				v.provider = append(v.provider, providerName)
				proxyFilterMap[v.Name] = v

				provider := map[string]any{
					"type":         "file",
					"path":         m["path"],
					"filter":       v.Filter,
					"interval":     interval,
					"health-check": defaultHealthCheck,
				}
				configMap["proxy-providers"].(map[string]any)[providerName] = provider
			}
		}
	}

	// add proxy groups
	if _, ok := configMap["proxy-groups"]; !ok {
		configMap["proxy-groups"] = []map[string]any{}
	}
	proxyGroups := configMap["proxy-groups"].([]map[string]any)
	// add default proxy groups
	proxyGroups = append(
		proxyGroups,
		map[string]any{
			"name": "Proxy",
			"type": "select",
			"use":  lo.Keys(airportFiles),
		},
	)
	// add airport proxy groups
	for name := range airportFiles {
		proxyGroups = append(
			proxyGroups,
			map[string]any{
				"name": name,
				"type": "select",
				"use":  []string{name},
			},
		)
	}
	// add custom proxy groups
	for _, v := range proxyConfig.Groups {
		// proxy 里添加机场
		if !v.RmAirports {
			v.Group.Proxies = append(v.Group.Proxies, lo.Keys(airportFiles)...)
		}

		// 替换相对应的 use_filters, us => us-sub.xsus.me, 不存在节点的自动跳过
		if len(v.UseFilters) > 0 {
			for _, filterName := range v.UseFilters {
				if _, exists := proxyFilterMap[filterName]; exists {
					v.Group.Use = append(v.Group.Use, proxyFilterMap[filterName].provider...)
				}
			}
		}

		var group map[string]any
		err := mapstructure.Decode(v.Group, &group)
		if err != nil {
			log.Fatalf("mapstructure.Decode(%s) err: %v", v.Group.Name, err)
			return
		}

		proxyGroups = append(proxyGroups, group)
	}

	proxyGroups = append(proxyGroups, map[string]any{
		"name":    "MATCH",
		"type":    "select",
		"proxies": []string{"Proxy"},
		"use":     lo.Keys(airportFiles),
	})
	configMap["proxy-groups"] = proxyGroups

	buf, err := yaml.Marshal(configMap)
	if err != nil {
		log.Fatal(err)
		return
	}

	err = safeWrite(filepath.Join(out, file), buf)
	if err != nil {
		log.Fatal(err)
		return
	}

	log.Println("finish")
}

func downloadProxies(rawURL string) (string, *ProxySchema, error) {
	uri, err := url.Parse(rawURL)
	if err != nil {
		return "", nil, err
	}

	buf, err := request(uri)
	if err != nil {
		return "", nil, err
	}

	schema := &ProxySchema{}
	if err := yaml.Unmarshal(buf, schema); err != nil {
		return "", nil, err
	}

	if schema.Proxies == nil {
		return "", nil, fmt.Errorf("file must have a `proxies` field (%s)", rawURL)
	}

	filename := fmt.Sprintf("%s.yaml", uri.Host)
	err = safeWrite(filepath.Join(out, filename), buf)
	if err != nil {
		return "", nil, fmt.Errorf("file to write %s (%s)", filepath.Join(out, filename), rawURL)
	}

	return filename, schema, nil
}

func request(uri *url.URL) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req := httpClient.R().SetContext(ctx)

	if user := uri.User; user != nil {
		password, _ := user.Password()
		req.SetBasicAuth(user.Username(), password)
	}

	res, err := req.Get(uri.String())
	if err != nil {
		return nil, err
	}
	return res.Body(), nil
}

func safeWrite(path string, buf []byte) error {
	dir := filepath.Dir(path)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, dirMode); err != nil {
			return err
		}
	}

	return os.WriteFile(path, buf, fileMode)
}

func unmarshal(path string, val interface{}) error {
	buf, err := readConfig(path)
	if err != nil {
		return fmt.Errorf("fail to readConfig %s: %v", path, err)
	}

	if err := yaml.Unmarshal(buf, val); err != nil {
		return fmt.Errorf("fail to yaml.Unmarshal %s: %v", path, err)
	}

	return nil
}

func readConfig(path string) ([]byte, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return nil, fmt.Errorf("configuration file %s is empty", path)
	}

	return data, err
}
