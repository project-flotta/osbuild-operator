package manifests_test

import (
	"context"
	"fmt"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/manifests"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuild"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildconfigtemplate"
	"github.com/project-flotta/osbuild-operator/tests/matchers"
)

var _ = Describe("OSBuild creation", func() {
	const (
		OSBuildConfigName = "osbuild-cfg"
	)
	var (
		ctx = context.TODO()

		osBuildConfig   v1alpha1.OSBuildConfig
		expectedOSBuild v1alpha1.OSBuild

		mockCtrl                        *gomock.Controller
		osBuildConfigRepository         *osbuildconfig.MockRepository
		configMapRepository             *configmap.MockRepository
		osBuildConfigTemplateRepository *osbuildconfigtemplate.MockRepository
		osBuildRepository               *osbuild.MockRepository
		scheme                          *runtime.Scheme

		creator *manifests.OSBuildCreator
	)

	BeforeEach(func() {
		scheme = runtime.NewScheme()
		utilruntime.Must(clientgoscheme.AddToScheme(scheme))
		utilruntime.Must(v1alpha1.AddToScheme(scheme))

		osBuildConfig = v1alpha1.OSBuildConfig{
			ObjectMeta: metav1.ObjectMeta{
				Name:      OSBuildConfigName,
				Namespace: "a-namespace",
			},
			Spec: v1alpha1.OSBuildConfigSpec{
				Details: v1alpha1.BuildDetails{
					Distribution: "rhel-86",
					Customizations: &v1alpha1.Customizations{
						Packages: []string{"a", "b"},
						Users:    []v1alpha1.User{{Name: "user-a"}},
						Services: &v1alpha1.Services{
							Enabled:  []string{"en-a", "en-b"},
							Disabled: []string{"dis-1", "dis-2"},
						},
					},
					TargetImage: v1alpha1.TargetImage{
						Architecture:    "x86_64",
						TargetImageType: "edge-container",
						OSTree:          nil,
					},
				},
			},
			Status: v1alpha1.OSBuildConfigStatus{},
		}

		expectedOSBuild = v1alpha1.OSBuild{
			ObjectMeta: metav1.ObjectMeta{
				Name:      configName(OSBuildConfigName, 1),
				Namespace: osBuildConfig.Namespace,
			},
			Spec: v1alpha1.OSBuildSpec{
				Details:     osBuildConfig.Spec.Details,
				TriggeredBy: "UpdateCR",
			},
		}
		err := controllerutil.SetControllerReference(&osBuildConfig, &expectedOSBuild, scheme)
		Expect(err).ToNot(HaveOccurred())

		mockCtrl = gomock.NewController(GinkgoT())
		osBuildRepository = osbuild.NewMockRepository(mockCtrl)
		osBuildConfigTemplateRepository = osbuildconfigtemplate.NewMockRepository(mockCtrl)
		osBuildConfigRepository = osbuildconfig.NewMockRepository(mockCtrl)
		configMapRepository = configmap.NewMockRepository(mockCtrl)

		creator = manifests.NewOSBuildCRCreator(
			osBuildConfigRepository,
			osBuildRepository,
			scheme,
			osBuildConfigTemplateRepository,
			configMapRepository,
		)
	})

	AfterEach(func() {
		mockCtrl.Finish()
	})

	Context("without template", func() {
		It("should create first OSBuild", func() {
			// given
			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, cp, gomock.Any())

			osBuildRepository.EXPECT().Create(ctx, &expectedOSBuild)

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create subsequent OSBuild", func() {
			// given
			previousVersion := 10
			osBuildConfig.Status.LastVersion = &previousVersion

			cp := osBuildConfig.DeepCopy()
			nextVersion := 11
			cp.Status.LastVersion = &nextVersion
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, matchers.NewOSBuildConfigStatusMatcher(cp), gomock.Any())

			expectedOSBuild.Name = configName(OSBuildConfigName, nextVersion)

			osBuildRepository.EXPECT().Create(ctx, &expectedOSBuild)

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail on OSBuildConfig patch failure", func() {
			// given
			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, cp, gomock.Any()).Return(fmt.Errorf("boom"))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})

		It("should fail on OSBuild creation failure", func() {
			// given
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, gomock.Any(), gomock.Any())

			osBuildRepository.EXPECT().Create(ctx, &expectedOSBuild).Return(fmt.Errorf("boom"))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})
	})

	Context("with template", func() {
		const (
			templateName = "template-name"
		)
		var (
			kickstartTxt           = "kickstart-raw"
			kickstartMap           corev1.ConfigMap
			template               v1alpha1.OSBuildConfigTemplate
			expectedCustomizations v1alpha1.Customizations
		)

		BeforeEach(func() {
			osBuildConfig.Spec.Template = &v1alpha1.Template{
				OSBuildConfigTemplateRef: templateName,
			}

			templateCustomizations := v1alpha1.Customizations{
				Packages: []string{"tmpl"},
				Users:    []v1alpha1.User{{Name: "user-tmpl"}},
				Services: &v1alpha1.Services{
					Enabled:  []string{"en-tmpl"},
					Disabled: []string{"dis-tmpl"},
				},
			}

			template = v1alpha1.OSBuildConfigTemplate{
				ObjectMeta: metav1.ObjectMeta{
					Name:            templateName,
					Namespace:       osBuildConfig.Namespace,
					ResourceVersion: "tmpl-rv-1",
				},
				Spec: v1alpha1.OSBuildConfigTemplateSpec{
					Customizations: &templateCustomizations,
				},
			}
			configCustomizations := osBuildConfig.Spec.Details.Customizations
			expectedCustomizations.Packages = append(templateCustomizations.Packages, configCustomizations.Packages...)
			expectedCustomizations.Users = append(templateCustomizations.Users, configCustomizations.Users...)
			expectedCustomizations.Services = &v1alpha1.Services{
				Enabled:  append(templateCustomizations.Services.Enabled, configCustomizations.Services.Enabled...),
				Disabled: append(templateCustomizations.Services.Disabled, configCustomizations.Services.Disabled...),
			}
			expectedOSBuild.Spec.Details.Customizations = &expectedCustomizations

			kickstartMap = corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      configName(OSBuildConfigName, 1),
					Namespace: osBuildConfig.Namespace,
				},
				Data: map[string]string{"kickstart": kickstartTxt},
			}
		})

		DescribeTable("should create with no kickstart", func(iso *v1alpha1.IsoConfiguration) {
			// given
			template.Spec.Iso = iso
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			cp.Status.CurrentTemplateResourceVersion = &template.ResourceVersion
			cp.Status.LastTemplateResourceVersion = &template.ResourceVersion
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, matchers.NewOSBuildConfigStatusMatcher(cp), gomock.Any())

			osBuildRepository.EXPECT().Create(ctx, matchers.NewOSBuildMatcher(&expectedOSBuild))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).ToNot(HaveOccurred())
		},
			Entry("no ISO", nil),
			Entry("no ISO.Kickstart", &v1alpha1.IsoConfiguration{}),
			Entry("no raw or config map ref", &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					// nothing
				},
			}),
		)

		It("should create when target kickstart map already exists", func() {
			// given
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					Raw: &kickstartTxt,
				},
			}
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			configMapRepository.EXPECT().Read(ctx, kickstartMap.Name, osBuildConfig.Namespace).
				Return(&kickstartMap, nil)

			configMapRepository.EXPECT().Patch(ctx, gomock.Any(), gomock.Any())

			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			cp.Status.CurrentTemplateResourceVersion = &template.ResourceVersion
			cp.Status.LastTemplateResourceVersion = &template.ResourceVersion
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, matchers.NewOSBuildConfigStatusMatcher(cp), gomock.Any())

			expectedOSBuild.Spec.Kickstart = &v1alpha1.NameRef{Name: kickstartMap.Name}
			osBuildRepository.EXPECT().Create(ctx, matchers.NewOSBuildMatcher(&expectedOSBuild))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create with raw kickstart", func() {
			// given
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					Raw: &kickstartTxt,
				},
			}
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			// Kickstart ConfigMap doesn't exist
			configMapRepository.EXPECT().Read(ctx, kickstartMap.Name, osBuildConfig.Namespace).
				Return(nil, errors.NewNotFound(schema.GroupResource{}, templateName))

			configMapRepository.EXPECT().Create(ctx, &kickstartMap)

			configMapRepository.EXPECT().Patch(ctx, &kickstartMap, gomock.Any())

			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			cp.Status.CurrentTemplateResourceVersion = &template.ResourceVersion
			cp.Status.LastTemplateResourceVersion = &template.ResourceVersion
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, matchers.NewOSBuildConfigStatusMatcher(cp), gomock.Any())

			expectedOSBuild.Spec.Kickstart = &v1alpha1.NameRef{Name: kickstartMap.Name}
			osBuildRepository.EXPECT().Create(ctx, matchers.NewOSBuildMatcher(&expectedOSBuild))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should create with ConfigMap kickstart", func() {
			// given
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)
			kickstartTemplateCMName := "kickstart-tmpl-cm"
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					ConfigMapName: &kickstartTemplateCMName,
				},
			}

			kickstartTemplateCM := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kickstartTemplateCMName,
					Namespace: osBuildConfig.Namespace,
				},
				Data: map[string]string{
					"kickstart": kickstartTxt,
				},
			}
			configMapRepository.EXPECT().Read(ctx, kickstartTemplateCMName, osBuildConfig.Namespace).
				Return(&kickstartTemplateCM, nil)

			// Kickstart ConfigMap doesn't exist
			configMapRepository.EXPECT().Read(ctx, kickstartMap.Name, osBuildConfig.Namespace).
				Return(nil, errors.NewNotFound(schema.GroupResource{}, templateName))

			configMapRepository.EXPECT().Create(ctx, &kickstartMap)

			configMapRepository.EXPECT().Patch(ctx, &kickstartMap, gomock.Any())

			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			cp.Status.CurrentTemplateResourceVersion = &template.ResourceVersion
			cp.Status.LastTemplateResourceVersion = &template.ResourceVersion
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, matchers.NewOSBuildConfigStatusMatcher(cp), gomock.Any())

			expectedOSBuild.Spec.Kickstart = &v1alpha1.NameRef{Name: kickstartMap.Name}
			osBuildRepository.EXPECT().Create(ctx, matchers.NewOSBuildMatcher(&expectedOSBuild))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).ToNot(HaveOccurred())
		})

		It("should fail when template not found", func() {
			// given
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).
				Return(nil, errors.NewNotFound(schema.GroupResource{}, templateName))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})

		It("should fail when kickstart template config map not found", func() {
			// given
			kickstartTemplateCMName := "kickstart-tmpl-cm"
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					ConfigMapName: &kickstartTemplateCMName,
				},
			}

			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			// Kickstart ConfigMap retrieval fails
			configMapRepository.EXPECT().Read(ctx, kickstartTemplateCMName, osBuildConfig.Namespace).
				Return(nil, fmt.Errorf("boom"))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})

		It("should fail when target kickstart config map reading fails", func() {
			// given
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					Raw: &kickstartTxt,
				},
			}
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			// Kickstart ConfigMap doesn't exist
			configMapRepository.EXPECT().Read(ctx, kickstartMap.Name, osBuildConfig.Namespace).
				Return(nil, fmt.Errorf("boom"))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})

		It("should fail when target kickstart config map creation fails", func() {
			// given
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					Raw: &kickstartTxt,
				},
			}
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			// Kickstart ConfigMap doesn't exist
			configMapRepository.EXPECT().Read(ctx, kickstartMap.Name, osBuildConfig.Namespace).
				Return(nil, errors.NewNotFound(schema.GroupResource{}, templateName))

			configMapRepository.EXPECT().Create(ctx, &kickstartMap).Return(fmt.Errorf("boom"))

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})

		It("should fail when target kickstart config map patching fails", func() {
			// given
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					Raw: &kickstartTxt,
				},
			}
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)

			// Kickstart ConfigMap doesn't exist
			configMapRepository.EXPECT().Read(ctx, kickstartMap.Name, osBuildConfig.Namespace).
				Return(nil, errors.NewNotFound(schema.GroupResource{}, templateName))

			configMapRepository.EXPECT().Create(ctx, &kickstartMap)

			configMapRepository.EXPECT().Patch(ctx, &kickstartMap, gomock.Any()).Return(fmt.Errorf("boom"))

			expectedOSBuild.Spec.Kickstart = &v1alpha1.NameRef{Name: kickstartMap.Name}
			osBuildRepository.EXPECT().Create(ctx, matchers.NewOSBuildMatcher(&expectedOSBuild))

			cp := osBuildConfig.DeepCopy()
			one := 1
			cp.Status.LastVersion = &one
			cp.Status.CurrentTemplateResourceVersion = &template.ResourceVersion
			cp.Status.LastTemplateResourceVersion = &template.ResourceVersion
			osBuildConfigRepository.EXPECT().PatchStatus(ctx, matchers.NewOSBuildConfigStatusMatcher(cp), gomock.Any())

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})

		It("should fail creating with malformed template kickstart config map", func() {
			// given
			osBuildConfigTemplateRepository.EXPECT().Read(ctx, templateName, osBuildConfig.Namespace).Return(&template, nil)
			kickstartTemplateCMName := "kickstart-tmpl-cm"
			template.Spec.Iso = &v1alpha1.IsoConfiguration{
				Kickstart: &v1alpha1.KickstartFile{
					ConfigMapName: &kickstartTemplateCMName,
				},
			}

			kickstartTemplateCM := corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      kickstartTemplateCMName,
					Namespace: osBuildConfig.Namespace,
				},
				// no Data
			}
			configMapRepository.EXPECT().Read(ctx, kickstartTemplateCMName, osBuildConfig.Namespace).
				Return(&kickstartTemplateCM, nil)

			// when
			err := creator.Create(ctx, &osBuildConfig)

			//then
			Expect(err).To(HaveOccurred())
		})
	})
})

func configName(name string, version int) string {
	return fmt.Sprintf("%s-%d", name, version)
}
