module github.com/fornellas/resonance

go 1.24.4

replace gopkg.in/yaml.v3 v3.0.1 => github.com/fornellas/yaml v0.0.0-20230305130802-4e6a4cc61a50

tool (
	github.com/client9/misspell/cmd/misspell
	github.com/fornellas/rrb
	github.com/fzipp/gocyclo/cmd/gocyclo
	github.com/gordonklaus/ineffassign
	github.com/jandelgado/gcov2lcov
	github.com/rakyll/gotest
	github.com/yoheimuta/protolint/cmd/protolint
	golang.org/x/tools/cmd/godoc
	golang.org/x/tools/cmd/goimports
	golang.org/x/vuln/cmd/govulncheck
	google.golang.org/grpc/cmd/protoc-gen-go-grpc
	google.golang.org/protobuf/cmd/protoc-gen-go
	honnef.co/go/tools/cmd/staticcheck
)

require (
	al.essio.dev/pkg/shellescape v1.6.0
	github.com/fatih/color v1.18.0
	github.com/fornellas/slogxt v1.0.1
	github.com/gliderlabs/ssh v0.3.8
	github.com/kylelemons/godebug v1.1.0
	github.com/spf13/cobra v1.9.1
	github.com/spf13/pflag v1.0.6
	github.com/spf13/viper v1.20.1
	github.com/stretchr/testify v1.10.0
	golang.org/x/crypto v0.39.0
	golang.org/x/sys v0.33.0
	golang.org/x/term v0.32.0
	google.golang.org/grpc v1.73.0
	google.golang.org/protobuf v1.36.6
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/BurntSushi/toml v1.4.1-0.20240526193622-a339e1f7089c // indirect
	github.com/anmitsu/go-shlex v0.0.0-20200514113438-38f4b401e2be // indirect
	github.com/bmatcuk/doublestar/v4 v4.8.1 // indirect
	github.com/chavacava/garif v0.1.0 // indirect
	github.com/client9/misspell v0.3.4 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/fornellas/rrb v0.2.6 // indirect
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/fzipp/gocyclo v0.6.0 // indirect
	github.com/gertd/go-pluralize v0.2.1 // indirect
	github.com/go-viper/mapstructure/v2 v2.2.1 // indirect
	github.com/golang/protobuf v1.5.4 // indirect
	github.com/gordonklaus/ineffassign v0.1.0 // indirect
	github.com/hashicorp/go-hclog v1.6.3 // indirect
	github.com/hashicorp/go-plugin v1.6.1 // indirect
	github.com/hashicorp/yamux v0.1.1 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/jandelgado/gcov2lcov v1.1.1 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/mitchellh/go-testing-interface v0.0.0-20171004221916-a61a99592b77 // indirect
	github.com/oklog/run v1.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/prometheus/procfs v0.15.1 // indirect
	github.com/rakyll/gotest v0.0.6 // indirect
	github.com/sagikazarmark/locafero v0.9.0 // indirect
	github.com/sirupsen/logrus v1.9.3 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.14.0 // indirect
	github.com/spf13/cast v1.9.2 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/williammartin/subreaper v0.0.0-20181101193406-731d9ece6883 // indirect
	github.com/yoheimuta/go-protoparser/v4 v4.14.0 // indirect
	github.com/yoheimuta/protolint v0.53.0 // indirect
	github.com/yuin/goldmark v1.4.13 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/exp/typeparams v0.0.0-20250207012021-f9890c6ad9f3 // indirect
	golang.org/x/mod v0.25.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sync v0.15.0 // indirect
	golang.org/x/telemetry v0.0.0-20240522233618-39ace7a40ae7 // indirect
	golang.org/x/text v0.26.0 // indirect
	golang.org/x/tools v0.33.0 // indirect
	golang.org/x/vuln v1.1.4 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250603155806-513f23925822 // indirect
	google.golang.org/grpc/cmd/protoc-gen-go-grpc v1.5.1 // indirect
	gopkg.in/tomb.v1 v1.0.0-20141024135613-dd632973f1e7 // indirect
	gopkg.in/yaml.v2 v2.4.0 // indirect
	honnef.co/go/tools v0.6.1 // indirect
)
