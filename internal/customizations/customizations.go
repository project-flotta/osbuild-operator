package customizations

import (
	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
)

var void struct{}

func MergeCustomizations(templateCustomizations, configCustomizations *v1alpha1.Customizations) *v1alpha1.Customizations {
	var customizations *v1alpha1.Customizations
	if configCustomizations != nil {
		customizations = configCustomizations.DeepCopy()
	}

	if templateCustomizations != nil {
		customizations = templateCustomizations.DeepCopy()
		if configCustomizations != nil {
			if configCustomizations.Services != nil {
				customizations.Services = mergeServices(templateCustomizations.Services, configCustomizations.Services)
			}
			if configCustomizations.Packages != nil {
				customizations.Packages = mergePackages(templateCustomizations.Packages, configCustomizations.Packages)
			}
			if configCustomizations.Users != nil {
				customizations.Users = mergeUsers(templateCustomizations.Users, configCustomizations.Users)
			}
		}
	}
	return customizations
}

func mergeServices(templateServices *v1alpha1.Services, configServices *v1alpha1.Services) *v1alpha1.Services {
	var enabled, disabled []string

	enabledSet := make(map[string]string)
	disabledSet := make(map[string]string)
	if templateServices != nil {
		for _, service := range templateServices.Enabled {
			enabledSet[service] = "template"
		}
		for _, service := range templateServices.Disabled {
			disabledSet[service] = "template"
		}
	}

	for _, service := range configServices.Enabled {
		enabledSet[service] = "config"
	}
	for _, service := range configServices.Disabled {
		disabledSet[service] = "config"
	}

	for service := range disabledSet {
		if source, ok := enabledSet[service]; ok {
			switch source {
			case "config":
				delete(disabledSet, service)
			case "template":
				delete(enabledSet, service)
			}
		}
	}
	for service := range enabledSet {
		enabled = append(enabled, service)
	}
	for service := range disabledSet {
		disabled = append(disabled, service)
	}
	services := v1alpha1.Services{Enabled: enabled, Disabled: disabled}
	return &services
}

func mergePackages(templatePackages []string, configPackages []string) []string {
	packagesSet := make(map[string]struct{})
	for _, pkg := range templatePackages {
		packagesSet[pkg] = void
	}
	for _, pkg := range configPackages {
		packagesSet[pkg] = void
	}
	var packages []string
	for pkg := range packagesSet {
		packages = append(packages, pkg)
	}
	return packages
}

func mergeUsers(templateUsers []v1alpha1.User, configUsers []v1alpha1.User) []v1alpha1.User {
	userIndex := make(map[string]v1alpha1.User)
	for _, user := range templateUsers {
		userIndex[user.Name] = user
	}
	for _, user := range configUsers {
		userIndex[user.Name] = user
	}
	var users []v1alpha1.User
	for _, user := range userIndex {
		users = append(users, user)
	}
	return users
}
