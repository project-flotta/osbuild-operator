package predicates_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	batchv1 "k8s.io/api/batch/v1"

	"github.com/project-flotta/osbuild-operator/internal/predicates"
)

var _ = Describe("OSBuildEnvConfig Job reconciliation predicate", func() {
	DescribeTable("should reconcile update", func(old, new runtimeclient.Object) {
		// given
		p := predicates.OSBuildEnvConfigJobFinished{}
		e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}

		// when
		shouldReconcile := p.Update(e)

		// then
		Expect(shouldReconcile).To(BeTrue())
	},
		Entry("when there are no active pods",
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 0,
				},
			},
		),
	)

	DescribeTable("should not reconcile update", func(old, new runtimeclient.Object) {
		// given
		p := predicates.OSBuildEnvConfigJobFinished{}
		e := event.UpdateEvent{ObjectOld: old, ObjectNew: new}

		// when
		shouldReconcile := p.Update(e)

		// then
		Expect(shouldReconcile).To(BeFalse())
	},
		Entry("when there are still active pods",
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
		),
		Entry("when old is missing",
			nil,
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
		),
		Entry("when new is missing",
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
			nil,
		),
		Entry("when new is not Job",
			&batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
			&batchv1.CronJob{},
		),
	)

	It("should not reconcile create", func() {
		// given
		p := predicates.OSBuildEnvConfigJobFinished{}
		e := event.CreateEvent{
			Object: &batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
		}

		// when
		shouldReconcile := p.Create(e)

		// then
		Expect(shouldReconcile).To(BeFalse())
	})

	It("should not reconcile delete", func() {
		// given
		p := predicates.OSBuildEnvConfigJobFinished{}
		e := event.DeleteEvent{
			Object: &batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
		}

		// when
		shouldReconcile := p.Delete(e)

		// then
		Expect(shouldReconcile).To(BeFalse())
	})

	It("should not reconcile for generic event", func() {
		// given
		// given
		p := predicates.OSBuildEnvConfigJobFinished{}
		e := event.GenericEvent{
			Object: &batchv1.Job{
				Status: batchv1.JobStatus{
					Active: 1,
				},
			},
		}

		// when
		shouldReconcile := p.Generic(e)

		// then
		Expect(shouldReconcile).To(BeFalse())
	})
})
