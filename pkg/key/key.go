package key

import (
	"fmt"

	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
)

func CFStackName(cluster v1alpha3.AWSCluster) string {
	return fmt.Sprintf("cluster-%s-tccpf", cluster.Name)
}
