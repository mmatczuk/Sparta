package sparta

import (
	"errors"
	"fmt"

	"github.com/Sirupsen/logrus"
	gocf "github.com/mweagle/go-cloudformation"
)

// cloudFormationAPIGatewayResource is the CustomResource type used to
// provision an APIGateway
type cloudFormationAPIGatewayResource struct {
	gocf.CloudFormationCustomResource
	ServiceToken *gocf.StringExpr
	API          interface{}
}

type cloudFormationS3PermissionResource struct {
	gocf.CloudFormationCustomResource
	ServiceToken *gocf.StringExpr
	Permission   interface{}
	LambdaTarget *gocf.StringExpr
	BucketArn    *gocf.StringExpr
}

type cloudFormationSNSPermissionResource struct {
	gocf.CloudFormationCustomResource
	ServiceToken *gocf.StringExpr
	Mode         string
	TopicArn     *gocf.StringExpr
	LambdaTarget *gocf.StringExpr
}

type cloudFormationSESPermissionResource struct {
	gocf.CloudFormationCustomResource
	ServiceToken *gocf.StringExpr
	Rules        interface{}
}

type cloudformationS3SiteManager struct {
	gocf.CloudFormationCustomResource
	ServiceToken *gocf.StringExpr
	TargetBucket *gocf.StringExpr
	SourceKey    *gocf.StringExpr
	SourceBucket *gocf.StringExpr
	APIGateway   map[string]*gocf.Output
}

func customTypeProvider(resourceType string) gocf.ResourceProperties {
	switch resourceType {
	case "Custom::SpartaAPIGateway":
		{
			return &cloudFormationAPIGatewayResource{}
		}
	case "Custom::SpartaS3Permission":
		{
			return &cloudFormationS3PermissionResource{}
		}
	case "Custom::SpartaSNSPermission":
		{
			return &cloudFormationSNSPermissionResource{}
		}
	case "Custom::SpartaSESPermission":
		{
			return &cloudFormationSESPermissionResource{}
		}
	case "Custom::SpartaS3SiteManager":
		{
			return &cloudformationS3SiteManager{}
		}
	default:
		return nil
	}
}

func init() {
	gocf.RegisterCustomResourceProvider(customTypeProvider)
}

func newCloudFormationResource(resourceType string, logger *logrus.Logger) (gocf.ResourceProperties, error) {
	resProps := gocf.NewResourceByType(resourceType)
	if nil == resProps {
		logger.WithFields(logrus.Fields{
			"Type": resourceType,
		}).Fatal("Failed to create CloudFormation CustomResource!")
		return nil, fmt.Errorf("Unsupported CustomResourceType: %s", resourceType)
	}
	return resProps, nil
}

func outputsForResource(template *gocf.Template, logicalResourceName string, logger *logrus.Logger) (map[string]interface{}, error) {
	item, ok := template.Resources[logicalResourceName]
	if !ok {
		return nil, nil
	}

	outputs := make(map[string]interface{}, 0)
	switch item.Properties.ResourceType() {
	case "AWS::S3::Bucket":
		outputs["Name"] = gocf.Ref(logicalResourceName).String()
		outputs["DomainName"] = gocf.GetAtt(logicalResourceName, "DomainName")
		outputs["WebsiteURL"] = gocf.GetAtt(logicalResourceName, "WebsiteURL")
		outputs["Type"] = gocf.String("AWS::S3::Bucket")
	}
	return outputs, nil

}
func safeAppendDependency(resource *gocf.Resource, dependencyName string) {
	if nil == resource.DependsOn {
		resource.DependsOn = []string{}
	}
	resource.DependsOn = append(resource.DependsOn, dependencyName)
}
func safeMetadataInsert(resource *gocf.Resource, key string, value interface{}) {
	if nil == resource.Metadata {
		resource.Metadata = make(map[string]interface{}, 0)
	}
	resource.Metadata[key] = value
}

func safeMergeTemplates(sourceTemplate *gocf.Template, destTemplate *gocf.Template, logger *logrus.Logger) error {
	var mergeErrors []string

	// Append the custom resources
	for eachKey, eachLambdaResource := range sourceTemplate.Resources {
		_, exists := destTemplate.Resources[eachKey]
		if exists {
			errorMsg := fmt.Sprintf("Duplicate CloudFormation resource name: %s", eachKey)
			mergeErrors = append(mergeErrors, errorMsg)
		} else {
			destTemplate.Resources[eachKey] = eachLambdaResource
		}
	}
	// Append the custom outputs
	for eachKey, eachLambdaOutput := range sourceTemplate.Outputs {
		_, exists := destTemplate.Outputs[eachKey]
		if exists {
			errorMsg := fmt.Sprintf("Duplicate CloudFormation output key name: %s", eachKey)
			mergeErrors = append(mergeErrors, errorMsg)
		} else {
			destTemplate.Outputs[eachKey] = eachLambdaOutput
		}
	}
	if len(mergeErrors) > 0 {
		logger.Error("Failed to update template. The following collisions were found:")
		for _, eachError := range mergeErrors {
			logger.Error("\t" + eachError)
		}
		return errors.New("Template merge failed")
	}
	return nil
}