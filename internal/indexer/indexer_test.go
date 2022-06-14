package indexer_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/indexer"
)

var _ = Describe("Index functions", func() {

	Context("Index func of os build config", func() {
		It("should create key", func() {
			// given
			templateRef := "template-ref"

			config := v1alpha1.OSBuildConfig{
				Spec: v1alpha1.OSBuildConfigSpec{
					Template: &v1alpha1.Template{
						OSBuildConfigTemplateRef: templateRef,
					},
				},
			}

			// when
			keys := indexer.ConfigByTemplateIndexFunc(&config)

			// then
			Expect(keys).To(HaveLen(1))
			Expect(keys).Should(ConsistOf(templateRef))
		})

		DescribeTable("should create no keys", func(template *v1alpha1.Template) {
			// given
			config := v1alpha1.OSBuildConfig{
				Spec: v1alpha1.OSBuildConfigSpec{
					Template: template,
				},
			}

			// when
			keys := indexer.ConfigByTemplateIndexFunc(&config)

			// then
			Expect(keys).To(BeEmpty())
		},
			Entry("no template", nil),
			Entry("no template ref", &v1alpha1.Template{}))

		It("should create no keys for wrong type", func() {
			// given
			notAConfig := v1alpha1.OSBuild{}

			// when
			keys := indexer.ConfigByTemplateIndexFunc(&notAConfig)

			// then
			Expect(keys).To(BeEmpty())
		})
	})

})
