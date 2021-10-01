module github.com/cortexlabs/cortex

go 1.14

require (
	cloud.google.com/go v0.81.0
	cloud.google.com/go/storage v1.12.0
	github.com/Azure/go-ansiterm v0.0.0-20170929234023-d6e3b3328b78 // indirect
	github.com/Microsoft/go-winio v0.4.11 // indirect
	github.com/StackExchange/wmi v0.0.0-20190523213315-cbe66965904d // indirect
	github.com/VojtechVitek/rerun v0.0.3 // indirect
	github.com/aws/aws-sdk-go v1.36.2
	github.com/bmizerany/assert v0.0.0-20160611221934-b7ed37b82869 // indirect
	github.com/containerd/containerd v1.4.3 // indirect
	github.com/cortexlabs/go-input v0.0.0-20200503032952-8b67a7a7b28d
	github.com/cortexlabs/yaml v0.0.0-20200511220111-581aea36a2e4
	github.com/cpuguy83/go-md2man/v2 v2.0.1 // indirect
	github.com/danwakefield/fnmatch v0.0.0-20160403171240-cbb64ac3d964 // indirect
	github.com/davecgh/go-spew v1.1.1
	github.com/denormal/go-gitignore v0.0.0-20180930084346-ae8ad1d07817
	github.com/docker/distribution v2.7.1+incompatible // indirect
	github.com/docker/docker v0.0.0-00010101000000-000000000000
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/docker/go-units v0.4.0 // indirect
	github.com/fatih/color v1.10.0
	github.com/fsnotify/fsnotify v1.5.1 // indirect
	github.com/getsentry/sentry-go v0.8.0
	github.com/go-delve/delve v1.7.2 // indirect
	github.com/go-ole/go-ole v1.2.4 // indirect
	github.com/gobwas/glob v0.2.3
	github.com/google/go-dap v0.6.0 // indirect
	github.com/google/uuid v1.1.2
	github.com/googleapis/gnostic v0.2.0 // indirect
	github.com/gorilla/mux v1.8.0
	github.com/gorilla/websocket v1.4.2
	github.com/imdario/mergo v0.3.6 // indirect
	github.com/kyoh86/looppointer v0.1.7 // indirect
	github.com/kyoh86/nolint v0.0.1 // indirect
	github.com/mattn/go-colorable v0.1.11 // indirect
	github.com/mattn/go-runewidth v0.0.13 // indirect
	github.com/mitchellh/go-homedir v1.1.0
	github.com/morikuni/aec v1.0.0 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.0.1 // indirect
	github.com/peterh/liner v1.2.1 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prometheus/client_golang v0.9.3
	github.com/prometheus/common v0.4.0
	github.com/segmentio/backo-go v0.0.0-20200129164019-23eae7c10bd3 // indirect
	github.com/shirou/gopsutil v3.20.11+incompatible
	github.com/sirupsen/logrus v1.8.1 // indirect
	github.com/spf13/cobra v1.2.1
	github.com/spf13/pflag v1.0.5
	github.com/stretchr/testify v1.7.0
	github.com/ugorji/go/codec v1.2.1
	github.com/xlab/treeprint v1.0.0
	github.com/xtgo/uuid v0.0.0-20140804021211-a0b114877d4c // indirect
	go.starlark.net v0.0.0-20210901212718-87f333178d59 // indirect
	go.uber.org/zap v1.17.0
	golang.org/x/arch v0.0.0-20210923205945-b76863e36670 // indirect
	golang.org/x/lint v0.0.0-20210508222113-6edffad5e616 // indirect
	golang.org/x/mod v0.5.1 // indirect
	golang.org/x/oauth2 v0.0.0-20210402161424-2e8d93401602
	golang.org/x/sys v0.0.0-20210930212924-f542c8878de8 // indirect
	golang.org/x/tools v0.1.7 // indirect
	google.golang.org/api v0.44.0
	google.golang.org/genproto v0.0.0-20210602131652-f16073e35f0c
	google.golang.org/grpc v1.38.0
	gopkg.in/check.v1 v1.0.0-20190902080502-41f04d3bba15 // indirect
	gopkg.in/karalabe/cookiejar.v2 v2.0.0-20150724131613-8dcd6a7f4951
	gopkg.in/segmentio/analytics-go.v3 v3.1.0
	gotest.tools v2.2.0+incompatible // indirect
	istio.io/api v0.0.0-20200911191701-0dc35ad5c478
	istio.io/client-go v0.0.0-20200807182027-d287a5abb594
	k8s.io/api v0.18.1
	k8s.io/apimachinery v0.18.1
	k8s.io/client-go v0.18.1
)

replace github.com/docker/docker => github.com/docker/engine v17.12.0-ce-rc1.0.20200618181300-9dc6525e6118+incompatible
