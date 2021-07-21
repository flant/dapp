module github.com/werf/werf

go 1.14

require (
	bou.ke/monkey v1.0.1
	github.com/Masterminds/goutils v1.1.1
	github.com/Masterminds/semver v1.5.0
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/acarl005/stripansi v0.0.0-20180116102854-5a71ef0e047d // indirect
	github.com/agl/ed25519 v0.0.0-20170116200512-5312a6153412 // indirect
	github.com/alessio/shellescape v0.0.0-20190409004728-b115ca0f9053
	github.com/aws/aws-sdk-go v1.35.20
	github.com/bitly/go-hostpool v0.1.0 // indirect
	github.com/bmatcuk/doublestar v1.1.5
	github.com/bugsnag/bugsnag-go v1.5.3 // indirect
	github.com/bugsnag/panicwrap v1.2.0 // indirect
	github.com/cloudflare/cfssl v1.4.1 // indirect
	github.com/containerd/containerd v1.5.0-beta.4 // indirect
	github.com/docker/cli v20.10.5+incompatible
	github.com/docker/distribution v2.7.1+incompatible
	github.com/docker/docker v20.10.5+incompatible
	github.com/docker/go v1.5.1-1 // indirect
	github.com/docker/go-connections v0.4.0
	github.com/docker/go-metrics v0.0.1 // indirect
	github.com/docker/go-units v0.4.0
	github.com/dustin/go-humanize v1.0.0
	github.com/emicklei/go-restful v2.13.0+incompatible // indirect
	github.com/ghodss/yaml v1.0.1-0.20190212211648-25d852aebe32
	github.com/go-git/go-billy/v5 v5.0.0
	github.com/go-git/go-git/v5 v5.1.1-0.20200721083337-cded5b685b8a
	github.com/go-openapi/spec v0.19.5
	github.com/go-openapi/strfmt v0.19.5
	github.com/go-openapi/validate v0.19.8
	github.com/gofrs/uuid v3.3.0+incompatible // indirect
	github.com/gogo/googleapis v1.4.0 // indirect
	github.com/google/go-containerregistry v0.2.0
	github.com/google/uuid v1.1.2
	github.com/gookit/color v1.3.7
	github.com/gosuri/uitable v0.0.4
	github.com/hailocab/go-hostpool v0.0.0-20160125115350-e80d13ce29ed // indirect
	github.com/hashicorp/go-multierror v1.0.0
	github.com/hashicorp/go-version v1.2.0
	github.com/helm/helm-2to3 v0.8.1
	github.com/huandu/xstrings v1.3.2 // indirect
	github.com/jinzhu/gorm v1.9.12 // indirect
	github.com/kardianos/osext v0.0.0-20190222173326-2bc1f35cddc0 // indirect
	github.com/miekg/pkcs11 v1.0.3 // indirect
	github.com/minio/minio v0.0.0-20210311070216-f92b7a562103
	github.com/mitchellh/copystructure v1.1.1
	github.com/moby/buildkit v0.8.2
	github.com/moby/sys/symlink v0.1.0 // indirect
	github.com/mvdan/xurls v1.1.0 // indirect
	github.com/oleiade/reflections v1.0.0 // indirect
	github.com/onsi/ginkgo v1.12.0
	github.com/onsi/gomega v1.9.0
	github.com/otiai10/copy v1.0.1
	github.com/otiai10/curr v1.0.0 // indirect
	github.com/pkg/errors v0.9.1
	github.com/prashantv/gostub v1.0.0
	github.com/rodaine/table v1.0.0
	github.com/satori/go.uuid v1.2.0
	github.com/sirupsen/logrus v1.8.1
	github.com/spaolacci/murmur3 v1.1.0
	github.com/spf13/cobra v1.1.3
	github.com/spf13/pflag v1.0.5
	github.com/theupdateframework/notary v0.6.1 // indirect
	github.com/tonistiigi/go-rosetta v0.0.0-20200727161949-f79598599c5d // indirect
	github.com/werf/kubedog v0.5.2-0.20210702152655-233cb8dec0be
	github.com/werf/lockgate v0.0.0-20200729113342-ec2c142f71ea
	github.com/werf/logboek v0.5.4
	github.com/xeipuuv/gojsonpointer v0.0.0-20190905194746-02993c407bfb // indirect
	github.com/xeipuuv/gojsonschema v1.2.0 // indirect
	golang.org/x/crypto v0.0.0-20210220033148-5ea612d1eb83
	golang.org/x/net v0.0.0-20210224082022-3d97a244fca7
	gopkg.in/dancannon/gorethink.v3 v3.0.5 // indirect
	gopkg.in/fatih/pool.v2 v2.0.0 // indirect
	gopkg.in/gorethink/gorethink.v3 v3.0.5 // indirect
	gopkg.in/ini.v1 v1.57.0
	gopkg.in/oleiade/reflections.v1 v1.0.0
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.0-20200615113413-eeeca48fe776 // indirect
	helm.sh/helm/v3 v3.5.1
	k8s.io/api v0.21.0
	k8s.io/apimachinery v0.21.0
	k8s.io/cli-runtime v0.21.0
	k8s.io/client-go v0.21.0
	k8s.io/helm v2.17.0+incompatible
	k8s.io/klog v1.0.0
	k8s.io/klog/v2 v2.8.0
	k8s.io/kubectl v0.21.0
	mvdan.cc/xurls v1.1.0
	rsc.io/letsencrypt v0.0.3 // indirect
	sigs.k8s.io/yaml v1.2.1-0.20210128145534-11e43d4a8b92
	vbom.ml/util v0.0.0-20180919145318-efcd4e0f9787 // indirect
)

replace github.com/jaguilar/vt100 => github.com/tonistiigi/vt100 v0.0.0-20190402012908-ad4c4a574305

replace github.com/Azure/go-autorest => github.com/Azure/go-autorest v13.3.2+incompatible

replace k8s.io/helm => github.com/werf/helm v0.0.0-20210202111118-81e74d46da0f

replace github.com/deislabs/oras => github.com/werf/oras v0.8.2-0.20210128161614-26d583f169ea

replace helm.sh/helm/v3 => github.com/werf/helm/v3 v3.0.0-20210625074159-a5d77a6118d1
