package customizations_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/customizations"
)

var (
	userA = v1alpha1.User{Name: "A", Groups: &[]string{"a"}}
	userB = v1alpha1.User{Name: "B", Groups: &[]string{"b1", "b2"}}
	userC = v1alpha1.User{Name: "C"}

	noUsers []v1alpha1.User

	emptyServices v1alpha1.Services

	enabledServices1  = []string{"a", "b"}
	disabledServices1 = []string{"c", "d"}

	enabledServices2  = []string{"aa", "bb"}
	disabledServices2 = []string{"cc", "dd"}

	enabledServices3  = append(enabledServices1, enabledServices2[0])
	disabledServices3 = append(disabledServices1, disabledServices2[0])
)

var _ = Describe("OSBuildConfig customizations", func() {
	DescribeTable("packages should be merged", func(templatePackages, configPackages, expectedPackages []string) {
		// given
		templateCustomizations := v1alpha1.Customizations{Packages: templatePackages}
		configCustomizations := v1alpha1.Customizations{Packages: configPackages}

		// when
		merged := customizations.MergeCustomizations(&templateCustomizations, &configCustomizations)

		// then
		Expect(merged.Packages).To(ConsistOf(expectedPackages))
		Expect(merged.Services).To(BeNil())
		Expect(merged.Users).To(BeNil())
	},
		Entry("no packages anywhere", []string{}, []string{}, []string{}),
		Entry("packages only in template", []string{"a", "b"}, []string{}, []string{"a", "b"}),
		Entry("packages only in config", []string{}, []string{"a", "b"}, []string{"a", "b"}),
		Entry("same packages in both", []string{"a", "b"}, []string{"a", "b"}, []string{"a", "b"}),
		Entry("packages in both, disjoint configs", []string{"a", "b"}, []string{"c", "d"}, []string{"a", "b", "c", "d"}),
		Entry("packages in both, overlapping configs", []string{"a", "b"}, []string{"b", "c"}, []string{"a", "b", "c"}),
	)

	DescribeTable("users should be merged", func(templateUsers, configUsers, expectedUsers []v1alpha1.User) {
		// given
		templateCustomizations := v1alpha1.Customizations{Users: templateUsers}
		configCustomizations := v1alpha1.Customizations{Users: configUsers}

		// when
		merged := customizations.MergeCustomizations(&templateCustomizations, &configCustomizations)

		// then
		Expect(merged.Users).To(ConsistOf(expectedUsers))
		Expect(merged.Services).To(BeNil())
		Expect(merged.Packages).To(BeNil())
	},
		Entry("no users anywhere", []v1alpha1.User{}, noUsers, []v1alpha1.User{}),
		Entry("users only in template", []v1alpha1.User{userA, userB}, noUsers, []v1alpha1.User{userA, userB}),
		Entry("users only in config", noUsers, []v1alpha1.User{userA, userB}, []v1alpha1.User{userA, userB}),
		Entry("same users in both", []v1alpha1.User{userA, userB}, []v1alpha1.User{userA, userB}, []v1alpha1.User{userA, userB}),
		Entry("users in both, disjoint config", []v1alpha1.User{userA, userB}, []v1alpha1.User{userC}, []v1alpha1.User{userA, userB, userC}),
		Entry("users in both, overlapping config", []v1alpha1.User{userA, userB}, []v1alpha1.User{userB, userC}, []v1alpha1.User{userA, userB, userC}),
	)

	DescribeTable("services should be merged", func(templateServices, configServices, expectedServices v1alpha1.Services) {
		// given
		templateCustomizations := v1alpha1.Customizations{Services: &templateServices}
		configCustomizations := v1alpha1.Customizations{Services: &configServices}

		// when
		merged := customizations.MergeCustomizations(&templateCustomizations, &configCustomizations)

		// then
		Expect(merged.Services.Enabled).To(ConsistOf(expectedServices.Enabled))
		Expect(merged.Services.Disabled).To(ConsistOf(expectedServices.Disabled))
		Expect(merged.Users).To(BeNil())
		Expect(merged.Packages).To(BeNil())
	},
		Entry("no services enabled or disabled anywhere", emptyServices, emptyServices, emptyServices),

		Entry("services only in template",
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1},
			emptyServices,
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1}),
		Entry("services only in config",
			emptyServices,
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1},
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1}),
		Entry("same services in both",
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1},
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1},
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1}),

		Entry("services in both, disjoint config",
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices1},
			v1alpha1.Services{Enabled: enabledServices2, Disabled: disabledServices2},
			v1alpha1.Services{Enabled: append(enabledServices1, enabledServices2...), Disabled: append(disabledServices1, disabledServices2...)}),
		Entry("services in both, overlapping config",
			v1alpha1.Services{Enabled: enabledServices2, Disabled: disabledServices2},
			v1alpha1.Services{Enabled: enabledServices3, Disabled: disabledServices3},
			v1alpha1.Services{Enabled: append(enabledServices1, enabledServices2...), Disabled: append(disabledServices1, disabledServices2...)}),

		Entry("enabled services in template, disabled in config",
			v1alpha1.Services{Enabled: enabledServices1},
			v1alpha1.Services{Disabled: disabledServices2},
			v1alpha1.Services{Enabled: enabledServices1, Disabled: disabledServices2}),
		Entry("enabled services in config, disabled in template",
			v1alpha1.Services{Disabled: disabledServices1},
			v1alpha1.Services{Enabled: enabledServices2},
			v1alpha1.Services{Enabled: enabledServices2, Disabled: disabledServices1}),
		Entry("only enabled services in both",
			v1alpha1.Services{Enabled: enabledServices1},
			v1alpha1.Services{Enabled: enabledServices2},
			v1alpha1.Services{Enabled: append(enabledServices1, enabledServices2...)}),
		Entry("only disabled services in both",
			v1alpha1.Services{Disabled: disabledServices1},
			v1alpha1.Services{Disabled: disabledServices2},
			v1alpha1.Services{Disabled: append(disabledServices1, disabledServices2...)}),

		Entry("config has enabled template services disabled",
			v1alpha1.Services{Enabled: enabledServices1},
			v1alpha1.Services{Disabled: enabledServices1},
			v1alpha1.Services{Disabled: enabledServices1}),
		Entry("config has disabled template services enabled",
			v1alpha1.Services{Disabled: disabledServices1},
			v1alpha1.Services{Enabled: disabledServices1},
			v1alpha1.Services{Enabled: disabledServices1}),
		Entry("config has one enabled service from template disabled",
			v1alpha1.Services{Enabled: []string{"a", "b"}, Disabled: []string{"c", "d"}},
			v1alpha1.Services{Disabled: []string{"a"}},
			v1alpha1.Services{Enabled: enabledServices1[1:], Disabled: append(disabledServices1, enabledServices1[0])}),
		Entry("config has one disabled service from template enabled",
			v1alpha1.Services{Enabled: []string{"a", "b"}, Disabled: []string{"c", "d"}},
			v1alpha1.Services{Enabled: []string{"c"}},
			v1alpha1.Services{Enabled: []string{"a", "b", "c"}, Disabled: []string{"d"}}),
		Entry("config has one disabled template service enabled and one enabled disabled",
			v1alpha1.Services{Enabled: []string{"a", "b"}, Disabled: []string{"c", "d"}},
			v1alpha1.Services{Enabled: []string{"c"}, Disabled: []string{"a"}},
			v1alpha1.Services{Enabled: []string{"b", "c"}, Disabled: []string{"a", "d"}}),
	)

	It("should use config customizations when template customizations are nil", func() {
		// given
		configCustomizations := v1alpha1.Customizations{
			Users:    []v1alpha1.User{userA, userB},
			Packages: []string{"a", "b"},
			Services: &v1alpha1.Services{
				Enabled:  enabledServices1,
				Disabled: disabledServices1,
			},
		}

		// when
		merged := customizations.MergeCustomizations(nil, &configCustomizations)

		// then
		Expect(*merged).To(BeEquivalentTo(configCustomizations))
	})

	It("should use template customizations when config customizations are nil", func() {
		// given
		templateCustomizations := v1alpha1.Customizations{
			Users:    []v1alpha1.User{userA, userB},
			Packages: []string{"a", "b"},
			Services: &v1alpha1.Services{
				Enabled:  enabledServices1,
				Disabled: disabledServices1,
			},
		}

		// when
		merged := customizations.MergeCustomizations(&templateCustomizations, nil)

		// then
		Expect(*merged).To(BeEquivalentTo(templateCustomizations))
	})

	It("should merge full configurations", func() {
		// given
		templateCustomizations := v1alpha1.Customizations{
			Users:    []v1alpha1.User{userB, userC},
			Packages: []string{"c", "d"},
			Services: &v1alpha1.Services{
				Enabled:  enabledServices2,
				Disabled: disabledServices2,
			},
		}

		configCustomizations := v1alpha1.Customizations{
			Users:    []v1alpha1.User{userA, userB},
			Packages: []string{"a", "b"},
			Services: &v1alpha1.Services{
				Enabled:  enabledServices1,
				Disabled: disabledServices1,
			},
		}

		// when
		merged := customizations.MergeCustomizations(&templateCustomizations, &configCustomizations)

		// then
		Expect(merged.Packages).To(ConsistOf("a", "b", "c", "d"))
		Expect(merged.Users).To(ConsistOf(userA, userB, userC))
		Expect(merged.Services.Enabled).To(ConsistOf(append(enabledServices1, enabledServices2...)))
		Expect(merged.Services.Disabled).To(ConsistOf(append(disabledServices1, disabledServices2...)))
	})
})
