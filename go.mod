module github.com/fornellas/resonance

go 1.20

require (
	github.com/adrg/xdg v0.4.0
	github.com/fatih/color v1.14.1
	github.com/openconfig/goyang v1.2.0
	github.com/sergi/go-diff v1.3.1
	github.com/sirupsen/logrus v1.9.0
	github.com/spf13/cobra v1.6.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/inconshreveable/mousetrap v1.0.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	golang.org/x/sys v0.5.0 // indirect
)

replace gopkg.in/yaml.v3 v3.0.1 => github.com/fornellas/yaml v0.0.0-20230305130802-4e6a4cc61a50
