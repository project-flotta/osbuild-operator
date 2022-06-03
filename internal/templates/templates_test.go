package templates_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	api "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/templates"
)

var _ = Describe("Templates", func() {
	DescribeTable("should replace parameters", func(template string, expectedParameters []api.Parameter,
		values []api.ParameterValue, expectedResult string) {
		// when
		result, err := templates.Process(template, expectedParameters, values)

		// then
		Expect(err).NotTo(HaveOccurred())
		Expect(result).To(Equal(expectedResult))
	},
		Entry("No parameters", "No parametes, no, no!", nil, nil, "No parametes, no, no!"),
		Entry("String parameters", "String one: {{.one}} and two: {{.two}}",
			[]api.Parameter{
				{Name: "one", DefaultValue: "", Type: "string"},
				{Name: "two", DefaultValue: "", Type: "string"},
			},
			[]api.ParameterValue{
				{Name: "one", Value: "o-n-e"},
				{Name: "two", Value: "t-w-o"},
			},
			"String one: o-n-e and two: t-w-o"),
		Entry("Int parameters", "Int one: {{.one}} and two: {{.two}}",
			[]api.Parameter{
				{Name: "one", DefaultValue: "", Type: "int"},
				{Name: "two", DefaultValue: "", Type: "int"},
			},
			[]api.ParameterValue{
				{Name: "one", Value: "1"},
				{Name: "two", Value: "2"},
			},
			"Int one: 1 and two: 2"),
		Entry("Bool parameters", "Bool one: {{.one}} and two: {{.two}}",
			[]api.Parameter{
				{Name: "one", DefaultValue: "", Type: "bool"},
				{Name: "two", DefaultValue: "", Type: "bool"},
			},
			[]api.ParameterValue{
				{Name: "one", Value: "true"},
				{Name: "two", Value: "false"},
			},
			"Bool one: true and two: false"),
		Entry("Mixed parameters", "String one: {{.one}}, int two: {{.two}} and bool three: {{.three}}",
			[]api.Parameter{
				{Name: "one", DefaultValue: "", Type: "string"},
				{Name: "two", DefaultValue: "", Type: "int"},
				{Name: "three", DefaultValue: "", Type: "bool"},
			},
			[]api.ParameterValue{
				{Name: "one", Value: "o-n-e"},
				{Name: "two", Value: "2"},
				{Name: "three", Value: "true"},
			},
			"String one: o-n-e, int two: 2 and bool three: true"),
		Entry("Using default parameters", "String one: {{.one}}, int two: {{.two}} and bool three: {{.three}}",
			[]api.Parameter{
				{Name: "one", DefaultValue: "o-n-e", Type: "string"},
				{Name: "two", DefaultValue: "2", Type: "int"},
				{Name: "three", DefaultValue: "true", Type: "bool"},
			},
			nil,
			"String one: o-n-e, int two: 2 and bool three: true"),

		Entry("Parameter outside of expected parameters set", "String {{.unknown}}",
			nil,
			nil,
			"String <no value>"),
	)

	DescribeTable("should fail processing template", func(template string, expectedParameters []api.Parameter,
		values []api.ParameterValue) {
		// when
		_, err := templates.Process(template, expectedParameters, values)

		// then
		Expect(err).To(HaveOccurred())
	},
		Entry("Invalid parameter", "Param: {{.invalid}}",
			[]api.Parameter{
				{Name: "one", DefaultValue: "false", Type: "bool"},
				{Name: "two", DefaultValue: "0", Type: "int"},
			},
			[]api.ParameterValue{
				{Name: "one", Value: "invalid"},
				{Name: "two", Value: "invalid"},
			},
		),
		Entry("Invalid template", "{{range .}}{{else}}{{continue}}{{end}}", nil, nil),
	)
})
