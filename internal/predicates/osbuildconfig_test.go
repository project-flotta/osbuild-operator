package predicates_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	"github.com/project-flotta/osbuild-operator/api/v1alpha1"
	"github.com/project-flotta/osbuild-operator/internal/predicates"
)

var (
	version1 = "1"
	version2 = "2"
)

var _ = Describe("OSBuildConfig reconciliation predicate", func() {
	DescribeTable("should reconcile update", func(old, new runtimeclient.Object) {
		// given
		p := predicates.OSBuildConfigChangedPredicate{}
		e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}

		// when
		shouldReconcile := p.Update(e)

		// then
		Expect(shouldReconcile).To(BeTrue())
	},
		Entry("when generation changes",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 2},
			},
		),
		Entry("when template version changed",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
				Status: v1alpha1.OSBuildConfigStatus{
					LastTemplateResourceVersion:    &version1,
					CurrentTemplateResourceVersion: &version2,
				},
			},
		),
	)

	DescribeTable("should not reconcile update", func(old, new runtimeclient.Object) {
		// given
		p := predicates.OSBuildConfigChangedPredicate{}
		e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}

		// when
		shouldReconcile := p.Update(e)

		// then
		Expect(shouldReconcile).To(BeFalse())
	},
		Entry("when nothing changes",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
		),
		Entry("when last template version is missing",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
				Status: v1alpha1.OSBuildConfigStatus{
					CurrentTemplateResourceVersion: &version2,
				},
			},
		),
		Entry("when current template version is missing",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
				Status: v1alpha1.OSBuildConfigStatus{
					LastTemplateResourceVersion: &version1,
				},
			},
		),
		Entry("when old is missing",
			nil,
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
		),
		Entry("when new is missing",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			nil,
		),
		Entry("when new is not OSBuildConfig",
			&v1alpha1.OSBuildConfig{
				ObjectMeta: v1.ObjectMeta{Generation: 1},
			},
			&v1alpha1.OSBuild{},
		),
	)

	It("should reconcile create", func() {
		// given
		p := predicates.OSBuildConfigChangedPredicate{}
		e := event.CreateEvent{Object: &v1alpha1.OSBuildConfig{
			ObjectMeta: v1.ObjectMeta{Generation: 1},
		}}

		// when
		shouldReconcile := p.Create(e)

		// then
		Expect(shouldReconcile).To(BeTrue())
	})

	It("should reconcile delete", func() {
		// given
		p := predicates.OSBuildConfigChangedPredicate{}
		e := event.DeleteEvent{Object: &v1alpha1.OSBuildConfig{
			ObjectMeta: v1.ObjectMeta{Generation: 1},
		}}

		// when
		shouldReconcile := p.Delete(e)

		// then
		Expect(shouldReconcile).To(BeTrue())
	})

	It("should reconcile for generic event", func() {
		// given
		p := predicates.OSBuildConfigChangedPredicate{}
		e := event.GenericEvent{Object: &v1alpha1.OSBuildConfig{
			ObjectMeta: v1.ObjectMeta{Generation: 1},
		}}

		// when
		shouldReconcile := p.Generic(e)

		// then
		Expect(shouldReconcile).To(BeTrue())
	})
})
