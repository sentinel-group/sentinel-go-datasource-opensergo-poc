module github.com/sentinel-group/sentinel-go-datasource-opensergo-poc

go 1.14

require (
	github.com/alibaba/sentinel-golang v1.0.3
	github.com/alibaba/sentinel-golang/pkg/adapters/gin v0.0.0-20220520143125-31fdb144b82e
	github.com/gin-gonic/gin v1.8.1
	github.com/go-logr/logr v0.4.0
	github.com/pkg/errors v0.9.1
	github.com/stretchr/testify v1.7.1
	k8s.io/apimachinery v0.21.4
	k8s.io/client-go v0.21.4
	sigs.k8s.io/controller-runtime v0.9.7
)
