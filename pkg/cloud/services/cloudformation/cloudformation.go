package cloudformation

import (
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	templateparser "github.com/awslabs/goformation/v6"
	"github.com/awslabs/goformation/v6/intrinsics"
	"github.com/giantswarm/microerror"
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

func (s *Service) CheckStackContainsAtLeastOneRouteDefinition(stackName string) (bool, error) {
	resourcesTypeCount := map[string]int{}
	{
		output, err := s.cloudFormationClient.GetTemplate(&cloudformation.GetTemplateInput{
			StackName: &stackName,
		})
		if err != nil {
			return false, microerror.Mask(err)
		}

		template, err := templateparser.ParseYAMLWithOptions([]byte(*output.TemplateBody), &intrinsics.ProcessorOptions{})
		if err != nil {
			return false, microerror.Mask(err)
		}

		for _, resource := range template.Resources {
			resourcesTypeCount[resource.AWSCloudFormationType()] += 1
		}
	}

	// Count how many `AWS::EC2::Route` are in the template.
	return resourcesTypeCount["AWS::EC2::Route"] > 0, nil
}

func (s *Service) DeleteStack(stackName string) error {
	if !strings.HasSuffix(stackName, "-tccpf") {
		return microerror.Mask(fmt.Errorf("can't delete a cloudformation whose name does not end with '-tccpf'"))
	}
	s.logger.Info("Deleting stack")

	describe, err := s.cloudFormationClient.DescribeStacks(&cloudformation.DescribeStacksInput{
		StackName: &stackName,
	})
	if err != nil {
		return microerror.Mask(err)
	}
	for _, stack := range describe.Stacks {
		if *stack.StackStatus != cloudformation.StackStatusCreateComplete && *stack.StackStatus != cloudformation.StackStatusUpdateComplete {
			return microerror.Mask(fmt.Errorf("can only delete stacks that are either in state %q or %q", cloudformation.StackStatusCreateComplete, cloudformation.StackStatusUpdateComplete))
		}
	}

	f := false

	// Ensure termination protection is disabled.
	_, err = s.cloudFormationClient.UpdateTerminationProtection(&cloudformation.UpdateTerminationProtectionInput{
		EnableTerminationProtection: &f,
		StackName:                   &stackName,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	_, err = s.cloudFormationClient.DeleteStack(&cloudformation.DeleteStackInput{
		StackName: &stackName,
	})
	if err != nil {
		return microerror.Mask(err)
	}

	s.logger.Info("Deleted stack")

	return nil
}
