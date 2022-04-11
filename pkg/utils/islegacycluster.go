package utils

import (
	"github.com/blang/semver"
	"github.com/giantswarm/k8smetadata/pkg/label"
	"github.com/go-logr/logr"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
)

func IsLegacyCluster(logger logr.Logger, cluster capi.Cluster) bool {
	// Try to get release label.
	versionStr, ok := cluster.GetLabels()[label.ReleaseVersion]
	if !ok {
		logger.Info("Label %q was not found in Cluster CR.", label.ReleaseVersion)
		return false
	}

	version, err := semver.ParseTolerant(versionStr)
	if err != nil {
		logger.Info("Unable to parse release version %q.", versionStr)
		return false
	}

	firstCapiVersion := semver.MustParse("20.0.0-alpha1")

	if version.GE(firstCapiVersion) {
		return false
	}

	return true
}
