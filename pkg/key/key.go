package key

import (
	"fmt"

	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

func CFStackName(cluster capi.Cluster) string {
	return fmt.Sprintf("cluster-%s-tccpf", cluster.Name)
}
