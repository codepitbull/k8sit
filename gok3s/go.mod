module github.com/codepitbull/go3ks

go 1.16

require (
	github.com/docker/go-connections v0.4.0 // indirect
	github.com/gin-gonic/gin v1.6.2 // indirect
	github.com/google/uuid v1.2.0
	github.com/gorilla/context v1.1.1 // indirect
	github.com/stretchr/testify v1.7.0
	github.com/testcontainers/testcontainers-go v0.11.1
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.22.1
	k8s.io/apimachinery v0.22.1
	k8s.io/client-go v0.22.1
	sigs.k8s.io/kind v0.11.1
	sigs.k8s.io/yaml v1.2.0 // indirect
)

replace github.com/testcontainers/testcontainers-go => github.com/codepitbull/testcontainers-go v0.11.2-0.20210824193510-3792295fdefd
