package matchers

import (
	"fmt"
	"reflect"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/gomega"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
)

type osBuildConfigStatusMatcher struct {
	expected *v1alpha1.OSBuildConfig
}

func NewOSBuildConfigStatusMatcher(expected *v1alpha1.OSBuildConfig) gomock.Matcher {
	return osBuildConfigStatusMatcher{expected: expected}
}

func (o osBuildConfigStatusMatcher) Matches(other interface{}) bool {
	if o.expected == nil || other == nil {
		return reflect.DeepEqual(o.expected, other)
	}

	actual, ok := other.(*v1alpha1.OSBuildConfig)
	if !ok {
		return false
	}

	if !reflect.DeepEqual(actual.ObjectMeta, o.expected.ObjectMeta) {
		return false
	}

	return reflect.DeepEqual(actual.Status, o.expected.Status)
}

func (o osBuildConfigStatusMatcher) String() string {
	return fmt.Sprintf("is equal to %v (%T)", o.expected, o.expected)
}

type osBuildMatcher struct {
	expected *v1alpha1.OSBuild
}

func NewOSBuildMatcher(expected *v1alpha1.OSBuild) gomock.Matcher {
	return osBuildMatcher{expected: expected}
}

func (o osBuildMatcher) Matches(other interface{}) bool {
	if o.expected == nil || other == nil {
		return reflect.DeepEqual(o.expected, other)
	}

	actual, ok := other.(*v1alpha1.OSBuild)
	if !ok {
		return false
	}

	if !reflect.DeepEqual(actual.ObjectMeta, o.expected.ObjectMeta) {
		return false
	}

	//[ECOPROJECT-917] TODO: validate the kickstart file as part of EdgeInstallerDetails
	//if !reflect.DeepEqual(actual.Spec.EdgeInstallerDetails.Kickstart, o.expected.Spec.EdgeInstallerDetails.Kickstart) {
	//	return false
	//}

	if actual.Spec.TriggeredBy != o.expected.Spec.TriggeredBy {
		return false
	}

	return matchBuildDetails(*actual.Spec.Details, *o.expected.Spec.Details)
}

func (o osBuildMatcher) String() string {
	return fmt.Sprintf("is equal to %v (%T)", o.expected, o.expected)
}

func matchBuildDetails(actualDetails v1alpha1.BuildDetails, expectedDetails v1alpha1.BuildDetails) bool {
	if actualDetails.Distribution != expectedDetails.Distribution {
		return false
	}

	if !reflect.DeepEqual(actualDetails.TargetImage, expectedDetails.TargetImage) {
		return false
	}

	if actualDetails.Customizations == nil || expectedDetails.Customizations == nil {
		return reflect.DeepEqual(actualDetails.Customizations, expectedDetails.Customizations)
	}

	if ok, err := ConsistOf(expectedDetails.Customizations.Packages).Match(actualDetails.Customizations.Packages); !ok || err != nil {
		return false
	}

	if ok, err := ConsistOf(expectedDetails.Customizations.Users).Match(actualDetails.Customizations.Users); !ok || err != nil {
		return false
	}

	if expectedDetails.Customizations.Services == nil || actualDetails.Customizations.Services == nil {
		return reflect.DeepEqual(expectedDetails.Customizations.Services, actualDetails.Customizations.Services)
	}

	if ok, err := ConsistOf(expectedDetails.Customizations.Services.Enabled).Match(actualDetails.Customizations.Services.Enabled); !ok || err != nil {
		return false
	}

	if ok, err := ConsistOf(expectedDetails.Customizations.Services.Disabled).Match(actualDetails.Customizations.Services.Disabled); !ok || err != nil {
		return false
	}

	return true
}
