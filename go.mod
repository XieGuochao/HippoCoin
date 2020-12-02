module github.com/XieGuochao/HippoCoin

go 1.15

require (
	github.com/XieGuochao/HippoCoin/host v0.0.0-00010101000000-000000000000
	github.com/XieGuochao/HippoCoin/ui v0.0.0-00010101000000-000000000000
	github.com/withmandala/go-log v0.1.0
	gopkg.in/yaml.v2 v2.4.0
)

replace github.com/XieGuochao/HippoCoin/host => ./host

replace github.com/XieGuochao/HippoCoin/ui => ./ui
