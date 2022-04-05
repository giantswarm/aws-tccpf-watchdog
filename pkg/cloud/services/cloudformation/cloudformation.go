package cloudformation

import (
	"reflect"
	"sort"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	templateparser "github.com/awslabs/goformation/v6"
	"github.com/awslabs/goformation/v6/intrinsics"
	"github.com/go-logr/logr"
)

// Service holds a collection of interfaces.
type Service struct {
	cloudFormationClient cloudformation.CloudFormation
	logger               logr.Logger
}

// NewService returns a new service given the cloudFormation api client.
func NewService(logger logr.Logger, cloudFormationClient cloudformation.CloudFormation) *Service {
	return &Service{
		cloudFormationClient: cloudFormationClient,
		logger:               logger,
	}
}

func (s *Service) CheckStackHasAllResources(stackName string) (bool, error) {
	existingResources := []string{}
	// Get a list of all resources in status `CREATE_COMPLETE` in the stack
	{
		output, err := s.cloudFormationClient.DescribeStackResources(&cloudformation.DescribeStackResourcesInput{
			StackName: &stackName,
		})
		if err != nil {
			return false, err
		}

		for _, resource := range output.StackResources {
			if *resource.ResourceStatus == cloudformation.ResourceStatusCreateComplete || *resource.ResourceStatus == cloudformation.ResourceStatusUpdateComplete {
				existingResources = append(existingResources, *resource.LogicalResourceId)
			}
		}
	}

	wantedResources := []string{}
	{
		output, err := s.cloudFormationClient.GetTemplate(&cloudformation.GetTemplateInput{
			StackName: &stackName,
		})
		if err != nil {
			return false, err
		}

		template, err := templateparser.ParseYAMLWithOptions([]byte(*output.TemplateBody), &intrinsics.ProcessorOptions{})
		if err != nil {
			return false, err
		}

		for name, _ := range template.Resources {
			wantedResources = append(wantedResources, name)
		}
	}

	sort.Strings(wantedResources)
	sort.Strings(existingResources)

	if !reflect.DeepEqual(wantedResources, existingResources) {
		s.logger.Info("CF is not satisfied", "Wanted resources", wantedResources, "Existing resources", existingResources)

		return false, nil
	}

	return true, nil
}

func (s *Service) DeleteStack(stackName string) error {
	return nil
}
