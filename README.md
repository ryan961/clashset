# clashset

## 执行参数说明
```bash
✗ ./clashset -h
Usage of ./clashset:
  -dir string
        配置文件地址 📁 , 默认配置文件 template.yaml & proxy-filters.yaml 文件夹 (default "./")
  -file string
        导出配置文件名称 📃 (default "config.yaml")
  -interval uint
        proxy-providers 配置刷新间隔时长(s) (default 86400)
  -out string
        导出文件夹 📁 (default "out/")
  -urls string
        订阅链接地址 🔗 (多个订阅链接之间 "," 分隔)
```


## template.yaml 自定义配置模版

预先定义用户配置文件模版（除 `proxy-providers` 和 `proxy-groups`
的其他配置），脚本会根据订阅地址和规则生成填充 `proxy-providers`和 `proxy-groups` 配置。`rule-provider`、`rules`...相关配置自行在
template.yaml 中配置。

## configuration.yaml 脚本配置

* `filters`: `proxy-providers` 中根据 `filter` 生成的节点分组，机场间单独分开（例如：`name: jp-airport1`
  ），后面会填充到对应的 `use_filters` 中,
  对于 `filter` 正则匹配不存在节点的分组则会自动跳过。

```yaml
filters:
  - { name: "hk", filter: '香港' }
  - { name: "tw", filter: '台湾' }
  - { name: "jp", filter: '日本' }
```

* `groups`: `group` 为 `GroupCommonOption`
  的相关配置（详见：https://github.com/Dreamacro/clash/blob/7f1b50f4a70fe353b8ea5d31ecfd4c751b3f2eb0/adapter/outboundgroup/parser.go#L23,
  ）, `use_filters` 为所使用的 `filters` 中分组名称，`rm_airports` 表示是否移除 `proxies` 中的机场节点（默认会添加所有机场分组）。

```yaml
groups:
  - { group: { name: "Apple", type: select, proxies: [ "DIRECT", "Proxy" ] }, use_filters: [ "us","jp" ] }
  - { group: { name: "Amazon", type: select, proxies: [ "Proxy" ] }, use_filters: [ "us","sg","hk" ] }
  - { group: { name: "YouTube", type: url-test, proxies: [ "Proxy" ], url: 'http://www.gstatic.com/generate_204', interval: 300 }, rm_airports: true }
```