package controllers_test

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	buildv1 "github.com/openshift/api/build/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	routev1 "github.com/openshift/api/route/v1"
	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	kubevirtv1 "kubevirt.io/api/core/v1"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/gomega"

	"github.com/project-flotta/osbuild-operator/internal/conf"
	"github.com/project-flotta/osbuild-operator/internal/repository/certificate"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/deployment"
	"github.com/project-flotta/osbuild-operator/internal/repository/job"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildenvconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/route"
	"github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/internal/repository/service"
	"github.com/project-flotta/osbuild-operator/internal/repository/virtualmachine"
	"github.com/project-flotta/osbuild-operator/internal/sshkey"

	"github.com/project-flotta/osbuild-operator/controllers"
)

var _ = Describe("OSBuildEnvConfig Controller", func() {
	const (
		operatorNamespace        = "osbuild"
		caIssuerName             = "osbuild-issuer"
		dbSecretName             = "composer-db"
		instanceName             = "env"
		osBuildOperatorFinalizer = "osbuilder.project-flotta.io/osBuildOperatorFinalizer"
		composerCertificateName  = "composer-cert"
		templatesDir             = "../resources/templates"
		workerSetupImageName     = "quay.io/project-flotta/osbuild-operator-worker-setup:latest"

		genericS3CredsSecretName    = "genericS3CredsSecretName" // #nosec G101
		genericS3Region             = "us-east-1"
		genericS3Bucket             = "test-bucket"
		genericS3Endpoint           = "https://somewhere"
		genericS3CABundleSecretName = "genericS3CABundleSecretName" // #nosec G101

		internalBuilderName               = "internal-builder"
		internalBuilderArchitectureString = "x86_64"
		internalBuilderImageUrl           = "http://nexus-osbuild:8081/repository/disk-images/rhel-8.6-x86_64-kvm.qcow2"

		externalBuilderName             = "external-builder"
		externalBuilderAddress          = "192.168.1.5"
		externalBuilderUser             = "external-user"
		externalBuilderSSHKeySecretName = "exteranl-ssh-keys" // #nosec G101
	)

	var (
		mockCtrl *gomock.Controller

		scheme *runtime.Scheme

		osBuildEnvConfigRepository *osbuildenvconfig.MockRepository
		certificateRepository      *certificate.MockRepository
		configMapRepository        *configmap.MockRepository
		deploymentRepository       *deployment.MockRepository
		jobRepository              *job.MockRepository
		routeRepository            *route.MockRepository
		serviceRepository          *service.MockRepository
		secretRepository           *secret.MockRepository
		virtualMachineRepository   *virtualmachine.MockRepository
		sshKeyGenerator            *sshkey.MockSSHKeyGenerator

		reconciler     *controllers.OSBuildEnvConfigReconciler
		requestContext context.Context

		errNotFound error
		errFailed   error

		instance       osbuildv1alpha1.OSBuildEnvConfig
		ownerReference metav1.OwnerReference

		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name: instanceName,
			},
		}

		resultRequeue      = ctrl.Result{Requeue: true}
		resultQuickRequeue = ctrl.Result{RequeueAfter: time.Second}
		resultDone         = ctrl.Result{}
	)

	BeforeEach(func() {
		os.Setenv("WORKING_NAMESPACE", operatorNamespace)
		os.Setenv("CA_ISSUER_NAME", caIssuerName)
		os.Setenv("TEMPLATES_DIR", templatesDir)
		os.Setenv("WORKER_SETUP_IMAGE", workerSetupImageName)
		err := conf.Load()
		Expect(err).To(BeNil())

		mockCtrl = gomock.NewController(GinkgoT())

		osBuildEnvConfigRepository = osbuildenvconfig.NewMockRepository(mockCtrl)
		certificateRepository = certificate.NewMockRepository(mockCtrl)
		configMapRepository = configmap.NewMockRepository(mockCtrl)
		deploymentRepository = deployment.NewMockRepository(mockCtrl)
		jobRepository = job.NewMockRepository(mockCtrl)
		routeRepository = route.NewMockRepository(mockCtrl)
		serviceRepository = service.NewMockRepository(mockCtrl)
		secretRepository = secret.NewMockRepository(mockCtrl)
		virtualMachineRepository = virtualmachine.NewMockRepository(mockCtrl)
		sshKeyGenerator = sshkey.NewMockSSHKeyGenerator(mockCtrl)

		scheme = runtime.NewScheme()
		err = clientgoscheme.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = osbuildv1alpha1.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = certmanagerv1.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = routev1.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = kubevirtv1.AddToScheme(scheme)
		Expect(err).To(BeNil())

		reconciler = &controllers.OSBuildEnvConfigReconciler{
			Scheme:                     scheme,
			OSBuildEnvConfigRepository: osBuildEnvConfigRepository,
			CertificateRepository:      certificateRepository,
			ConfigMapRepository:        configMapRepository,
			DeploymentRepository:       deploymentRepository,
			JobRepository:              jobRepository,
			RouteRepository:            routeRepository,
			ServiceRepository:          serviceRepository,
			SecretRepository:           secretRepository,
			VirtualMachineRepository:   virtualMachineRepository,
			SSHKeyGenerator:            sshKeyGenerator,
		}

		requestContext = context.TODO()

		errNotFound = errors.NewNotFound(schema.GroupResource{}, "Requested resource was not found")
		errFailed = errors.NewInternalError(fmt.Errorf("Server encounter and error"))

		internalBuilderArchitecture := (osbuildv1alpha1.Architecture)(internalBuilderArchitectureString)
		instance = osbuildv1alpha1.OSBuildEnvConfig{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "osbuilder.project-flotta.io/v1alpha1",
				Kind:       "OSBuildEnvConfig",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: instanceName,
			},
			Spec: osbuildv1alpha1.OSBuildEnvConfigSpec{
				Composer: &osbuildv1alpha1.ComposerConfig{},
				Workers: osbuildv1alpha1.WorkersConfig{
					{
						Name: internalBuilderName,
						VMWorkerConfig: &osbuildv1alpha1.VMWorkerConfig{
							Architecture: &internalBuilderArchitecture,
							ImageURL:     internalBuilderImageUrl,
						},
					},
					{
						Name: externalBuilderName,
						ExternalWorkerConfig: &osbuildv1alpha1.ExternalWorkerConfig{
							Address: externalBuilderAddress,
							User:    externalBuilderUser,
							SSHKeySecretReference: buildv1.SecretLocalReference{
								Name: externalBuilderSSHKeySecretName,
							},
						},
					},
				},
				S3Service: osbuildv1alpha1.S3ServiceConfig{
					GenericS3: &osbuildv1alpha1.GenericS3ServiceConfig{
						AWSS3ServiceConfig: &osbuildv1alpha1.AWSS3ServiceConfig{
							CredsSecretReference: buildv1.SecretLocalReference{
								Name: genericS3CredsSecretName,
							},
							Region: genericS3Region,
							Bucket: genericS3Bucket,
						},
						Endpoint: genericS3Endpoint,
						CABundleSecretReference: &buildv1.SecretLocalReference{
							Name: genericS3CABundleSecretName,
						},
						SkipSSLVerification: pointer.Bool(true),
					},
				},
			},
		}

		ownerReference = metav1.OwnerReference{
			APIVersion:         instance.APIVersion,
			Kind:               instance.Kind,
			Name:               instance.Name,
			UID:                instance.UID,
			BlockOwnerDeletion: pointer.BoolPtr(true),
			Controller:         pointer.BoolPtr(true),
		}
	})

	AfterEach(func() {
		os.Unsetenv("WORKING_NAMESPACE")
		os.Unsetenv("CA_ISSUER_NAME")
		os.Unsetenv("TEMPLATES_DIR")
		mockCtrl.Finish()
	})

	Context("Failure to get instance", func() {
		It("Should return Done when the instance is not found", func() {
			// given
			osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(nil, errNotFound)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))

		})

		It("Should return error when failed to get the instance", func() {
			// given
			osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(nil, errFailed)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(Equal(errFailed))
			Expect(result).To(Equal(resultRequeue))
		})
	})

	Context("Handle deletion", func() {

		BeforeEach(func() {
			instance.ObjectMeta.DeletionTimestamp = &metav1.Time{Time: time.Now()}
		})

		AfterEach(func() {
			instance.ObjectMeta.DeletionTimestamp = nil
		})

		It("Should return Done if there is no Finalizer", func() {
			// given
			osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
			// when
			result, err := reconciler.Reconcile(requestContext, request)
			// then
			Expect(err).To(BeNil())
			Expect(result).To(Equal(resultDone))
		})

		Context("Handle cleanup", func() {
			var (
				originalInstance *osbuildv1alpha1.OSBuildEnvConfig
			)
			BeforeEach(func() {
				instance.ObjectMeta.Finalizers = []string{osBuildOperatorFinalizer}
				osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
				originalInstance = instance.DeepCopy()
			})

			AfterEach(func() {
				instance.ObjectMeta.Finalizers = nil
			})

			It("Should return error if failed to remove finalizer", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Patch(requestContext, originalInstance, &instance).Return(errFailed)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(Equal(errFailed))
				Expect(result).To(Equal(resultRequeue))
			})

			It("Should return Done if removed the finalizer successfully", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Patch(requestContext, originalInstance, &instance).Return(nil)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(BeNil())
				Expect(result).To(Equal(resultDone))
			})
		})
	})

	Context("Handle update", func() {
		BeforeEach(func() {
			osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
		})

		Context("Adding Finalizers", func() {
			var (
				originalInstance *osbuildv1alpha1.OSBuildEnvConfig
			)

			BeforeEach(func() {
				originalInstance = instance.DeepCopy()
			})

			It("Should return error if it failed to add the finalizer", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Patch(requestContext, originalInstance, &instance).Return(errFailed)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(Equal(errFailed))
				Expect(result).To(Equal(resultRequeue))
			})

			It("Should return requeue if the finalizer was added", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Patch(requestContext, originalInstance, &instance).Return(nil)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(BeNil())
				Expect(result).To(Equal(resultQuickRequeue))
			})
		})

		Context("With finalizer", func() {
			const (
				composerWorkerAPIRouteName   = "osbuild-worker"
				composerWorkerAPIServiceName = "osbuild-worker"
				composerWorkerAPIRouteHost   = "osbuild-worker.apps.my-cluster.example.com"
			)

			var (
				composerWorkerAPIRoute routev1.Route
			)

			BeforeEach(func() {
				instance.ObjectMeta.Finalizers = []string{osBuildOperatorFinalizer}

				composerWorkerAPIRoute = routev1.Route{
					ObjectMeta: metav1.ObjectMeta{
						Name:            composerWorkerAPIRouteName,
						Namespace:       conf.GlobalConf.WorkingNamespace,
						OwnerReferences: []metav1.OwnerReference{ownerReference},
					},
					Spec: routev1.RouteSpec{
						To: routev1.RouteTargetReference{
							Kind: "Service",
							Name: composerWorkerAPIServiceName,
						},
						TLS: &routev1.TLSConfig{
							Termination: routev1.TLSTerminationPassthrough,
						},
					},
				}

			})

			It("Should return an error if failed to get the Route for the Composer Worker API ", func() {
				// given
				routeRepository.EXPECT().Read(requestContext, composerWorkerAPIRouteName, operatorNamespace).Return(nil, errFailed)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(Equal(errFailed))
				Expect(result).To(Equal(resultRequeue))
			})

			Context("Route for the Composer Worker API not found", func() {

				BeforeEach(func() {
					routeRepository.EXPECT().Read(requestContext, composerWorkerAPIRouteName, operatorNamespace).Return(nil, errNotFound)
				})

				It("Should return an error if failed to create the Route for the Composer Worker API", func() {
					// given
					routeRepository.EXPECT().Create(requestContext, &composerWorkerAPIRoute).Return(errFailed)
					// when
					result, err := reconciler.Reconcile(requestContext, request)
					// then
					Expect(err).To(Equal(errFailed))
					Expect(result).To(Equal(resultRequeue))
				})

				It("Should return requeue if succeeded to create the Route for the Composer Worker API", func() {
					// given
					routeRepository.EXPECT().Create(requestContext, &composerWorkerAPIRoute).Return(nil)
					// when
					result, err := reconciler.Reconcile(requestContext, request)
					// then
					Expect(err).To(BeNil())
					Expect(result).To(Equal(resultQuickRequeue))
				})
			})

			Context("Route for the Composer Worker API exists ", func() {
				BeforeEach(func() {
					routeRepository.EXPECT().Read(requestContext, composerWorkerAPIRouteName, operatorNamespace).Return(&composerWorkerAPIRoute, nil)
					composerWorkerAPIRoute.Status.Ingress = []routev1.RouteIngress{
						{
							Conditions: []routev1.RouteIngressCondition{
								{
									Type: routev1.RouteAdmitted,
								},
							},
						},
					}
				})

				It("Should return an error if failed to check if the Route for the Composer Worker API is ready", func() {
					//given
					routeRepository.EXPECT().Read(requestContext, composerWorkerAPIRouteName, operatorNamespace).Return(nil, errFailed)
					// when
					result, err := reconciler.Reconcile(requestContext, request)
					// then
					Expect(err).To(Equal(errFailed))
					Expect(result).To(Equal(resultRequeue))

				})

				It("Should return requeue if the Route for the Composer Worker API is not ready", func() {
					// given
					composerWorkerAPIRoute.Status.Ingress[0].Conditions[0].Status = corev1.ConditionFalse
					routeRepository.EXPECT().Read(requestContext, composerWorkerAPIRouteName, operatorNamespace).Return(&composerWorkerAPIRoute, nil)
					// when
					result, err := reconciler.Reconcile(requestContext, request)
					// then
					Expect(err).To(BeNil())
					Expect(result).To(Equal(reconcile.Result{Requeue: true, RequeueAfter: time.Second * 10}))
				})

				Context("Worker API Route is ready", func() {
					const (
						certificateDuration            = 87600
						composerComposerAPIServiceName = "osbuild-composer"
					)
					var (
						composerCertificate certmanagerv1.Certificate
					)

					BeforeEach(func() {
						composerWorkerAPIRoute.Status.Ingress[0].Conditions[0].Status = corev1.ConditionTrue
						composerWorkerAPIRoute.Status.Ingress[0].Host = composerWorkerAPIRouteHost
						routeRepository.EXPECT().Read(requestContext, composerWorkerAPIRouteName, operatorNamespace).Return(&composerWorkerAPIRoute, nil)

						composerCertificate = certmanagerv1.Certificate{
							ObjectMeta: metav1.ObjectMeta{
								Name:            composerCertificateName,
								Namespace:       operatorNamespace,
								OwnerReferences: []metav1.OwnerReference{ownerReference},
							},
							Spec: certmanagerv1.CertificateSpec{
								SecretName: composerCertificateName,
								PrivateKey: &certmanagerv1.CertificatePrivateKey{
									Algorithm: "ECDSA",
									Size:      256,
								},
								DNSNames: []string{
									composerComposerAPIServiceName,
									composerWorkerAPIServiceName,
									composerWorkerAPIRouteHost,
								},
								Duration: &metav1.Duration{
									Duration: time.Hour * certificateDuration,
								},
								IssuerRef: certmanagermetav1.ObjectReference{
									Group: "cert-manager.io",
									Kind:  "Issuer",
									Name:  caIssuerName,
								},
							},
						}
					})

					AfterEach(func() {
						instance.ObjectMeta.Finalizers = nil
					})

					It("Should return an error if it failed to get the certificate", func() {
						// given
						certificateRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errFailed)
						// when
						result, err := reconciler.Reconcile(requestContext, request)
						// then
						Expect(err).To(Equal(errFailed))
						Expect(result).To(Equal(resultRequeue))
					})

					It("Should return an error if failed to create the certificate", func() {
						// given
						certificateRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errNotFound)
						certificateRepository.EXPECT().Create(requestContext, &composerCertificate).Return(errFailed)
						// when
						result, err := reconciler.Reconcile(requestContext, request)
						// then
						Expect(err).To(Equal(errFailed))
						Expect(result).To(Equal(resultRequeue))
					})

					It("Should return requeue if the certificate was created", func() {
						// given
						certificateRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errNotFound)
						certificateRepository.EXPECT().Create(requestContext, &composerCertificate).Return(nil)
						// when
						result, err := reconciler.Reconcile(requestContext, request)
						// then
						Expect(err).To(BeNil())
						Expect(result).To(Equal(resultQuickRequeue))
					})

					Context("Composer Certificate is already created", func() {
						var (
							composerCertificateSecret corev1.Secret
						)

						BeforeEach(func() {
							certificateRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(&composerCertificate, nil)

							composerCertificateSecret = corev1.Secret{
								ObjectMeta: metav1.ObjectMeta{
									Namespace: operatorNamespace,
									Name:      composerCertificateName,
								},
							}
						})

						It("Should return an error if it failed to get the composer certificate secret", func() {
							// given
							secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errFailed)
							// when
							result, err := reconciler.Reconcile(requestContext, request)
							// then
							Expect(err).To(Equal(errFailed))
							Expect(result).To(Equal(resultRequeue))
						})

						Context("Successfully get the composer certificate secret", func() {
							var (
								originalComposerCertificateSecret *corev1.Secret
								updatedComposerCertificateSecret  *corev1.Secret
							)

							BeforeEach(func() {
								secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(&composerCertificateSecret, nil)

								originalComposerCertificateSecret = composerCertificateSecret.DeepCopy()
								updatedComposerCertificateSecret = composerCertificateSecret.DeepCopy()
								updatedComposerCertificateSecret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerReference}
							})

							It("Should return an error if failed to update the secret owner", func() {
								// given
								secretRepository.EXPECT().Patch(requestContext, originalComposerCertificateSecret, updatedComposerCertificateSecret).Return(errFailed)
								// when
								result, err := reconciler.Reconcile(requestContext, request)
								// then
								Expect(err).To(Equal(errFailed))
								Expect(result).To(Equal(resultRequeue))
							})

							It("Should return requeue if succeeded to update the secret owner", func() {
								// given
								secretRepository.EXPECT().Patch(requestContext, originalComposerCertificateSecret, updatedComposerCertificateSecret).Return(nil)
								// when
								result, err := reconciler.Reconcile(requestContext, request)
								// then
								Expect(err).To(BeNil())
								Expect(result).To(Equal(resultQuickRequeue))
							})
						})

						Context("Composer Certificate Owner is already set", func() {
							BeforeEach(func() {
								composerCertificateSecret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerReference}
								secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(&composerCertificateSecret, nil)
							})

							It("Should return an error if PSQL information is not set", func() {
								// given
								psqlError := fmt.Errorf("creating a PSQL service is not yet implemented")
								// when
								result, err := reconciler.Reconcile(requestContext, request)
								// then
								Expect(err).To(Equal(psqlError))
								Expect(result).To(Equal(resultRequeue))

							})

							Context("PSQL information is set", func() {
								const (
									composerConfigMapName = "osbuild-composer-config"
								)
								BeforeEach(func() {
									var sslMode osbuildv1alpha1.DBSSLMode = "disable"

									instance.Spec.Composer.PSQL = &osbuildv1alpha1.ComposerDBConfig{
										ConnectionSecretReference: buildv1.SecretLocalReference{
											Name: dbSecretName,
										},
										SSLMode: &sslMode,
									}
								})

								It("Should return an error if failed to get the ConfigMap for the osbuild-composer configuration", func() {
									// given
									configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(nil, errFailed)
									// when
									result, err := reconciler.Reconcile(requestContext, request)
									// then
									Expect(err).To(Equal(errFailed))
									Expect(result).To(Equal(resultRequeue))
								})

								It("Should return an error if failed to create the ConfigMap for the osbuild-composer configuration", func() {
									// given
									configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(nil, errNotFound)
									configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
									// when
									result, err := reconciler.Reconcile(requestContext, request)
									// then
									Expect(err).To(Equal(errFailed))
									Expect(result).To(Equal(resultRequeue))
								})

								It("Should return requeue if the ConfigMap for the osbuild-composer configuration was created", func() {
									// given
									configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(nil, errNotFound)
									configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
									// when
									result, err := reconciler.Reconcile(requestContext, request)
									// then
									Expect(err).To(BeNil())
									Expect(result).To(Equal(resultQuickRequeue))
								})

								Context("ConfigMap for the Composer configuration exists", func() {
									const (
										composerProxyConfigMapName = "osbuild-composer-proxy-config"
									)
									var (
										composerConfigConfigMap = corev1.ConfigMap{
											ObjectMeta: metav1.ObjectMeta{
												Namespace: operatorNamespace,
												Name:      composerConfigMapName,
											},
										}
									)

									BeforeEach(func() {
										configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(&composerConfigConfigMap, nil)
									})

									It("Should return an error if failed to get the ConfigMap for the proxy configuration", func() {
										// given
										configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(nil, errFailed)
										// when
										result, err := reconciler.Reconcile(requestContext, request)
										// then
										Expect(err).To(Equal(errFailed))
										Expect(result).To(Equal(resultRequeue))
									})

									It("Should return an error if failed to create the ConfigMap for the proxy configuration", func() {
										// given
										configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(nil, errNotFound)
										configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
										// when
										result, err := reconciler.Reconcile(requestContext, request)
										// then
										Expect(err).To(Equal(errFailed))
										Expect(result).To(Equal(resultRequeue))
									})

									It("Should return requeue if the ConfigMap for the proxy configuration was created", func() {
										// given
										configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(nil, errNotFound)
										configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
										// when
										result, err := reconciler.Reconcile(requestContext, request)
										// then
										Expect(err).To(BeNil())
										Expect(result).To(Equal(resultQuickRequeue))
									})

									Context("ConfigMap for the Proxy configuration exists", func() {
										const (
											composerDeploymentName = "composer"
										)
										var (
											proxyConfigConfigMap = corev1.ConfigMap{
												ObjectMeta: metav1.ObjectMeta{
													Namespace: operatorNamespace,
													Name:      composerProxyConfigMapName,
												},
											}
										)

										BeforeEach(func() {
											configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(&proxyConfigConfigMap, nil)
										})

										It("Should return an error if failed to get the composer deployment", func() {
											// given
											deploymentRepository.EXPECT().Read(requestContext, composerDeploymentName, operatorNamespace).Return(nil, errFailed)
											// when
											result, err := reconciler.Reconcile(requestContext, request)
											// then
											Expect(err).To(Equal(errFailed))
											Expect(result).To(Equal(resultRequeue))
										})

										Context("Composer deployment not found", func() {
											BeforeEach(func() {
												deploymentRepository.EXPECT().Read(requestContext, composerDeploymentName, operatorNamespace).Return(nil, errNotFound)
											})

											It("Should return an error if failed to create the composer deployment", func() {
												// given
												deploymentRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
												// when
												result, err := reconciler.Reconcile(requestContext, request)
												// then
												Expect(err).To(Equal(errFailed))
												Expect(result).To(Equal(resultRequeue))
											})

											It("Should return requeue if the composer deployment was created", func() {
												// given
												deploymentRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
												// when
												result, err := reconciler.Reconcile(requestContext, request)
												// then
												Expect(err).To(BeNil())
												Expect(result).To(Equal(resultQuickRequeue))
											})
										})

										Context("Composer Deployment exists", func() {
											const (
												composerComposerAPIPortName = "composer-api"
											)
											var (
												composerDeployment = appsv1.Deployment{
													ObjectMeta: metav1.ObjectMeta{
														Namespace: operatorNamespace,
														Name:      composerDeploymentName,
													},
												}

												composerAPIExternalPort = intstr.FromInt(8080)

												composerComposerAPIService corev1.Service
											)

											BeforeEach(func() {
												deploymentRepository.EXPECT().Read(requestContext, composerDeploymentName, operatorNamespace).Return(&composerDeployment, nil)

												composerComposerAPIService = corev1.Service{
													ObjectMeta: metav1.ObjectMeta{
														Namespace:       operatorNamespace,
														Name:            composerComposerAPIServiceName,
														OwnerReferences: []metav1.OwnerReference{ownerReference},
													},
													Spec: corev1.ServiceSpec{
														Type: corev1.ServiceTypeClusterIP,
														Ports: []corev1.ServicePort{
															{
																Name:       composerComposerAPIPortName,
																Port:       443,
																Protocol:   "TCP",
																TargetPort: composerAPIExternalPort,
															},
														},
														Selector: map[string]string{
															"app": "osbuild-composer",
														},
													},
												}
											})

											It("Should return an error if failed to get the composer api service", func() {
												// given
												serviceRepository.EXPECT().Read(requestContext, composerComposerAPIServiceName, operatorNamespace).Return(nil, errFailed)
												// when
												result, err := reconciler.Reconcile(requestContext, request)
												// then
												Expect(err).To(Equal(errFailed))
												Expect(result).To(Equal(resultRequeue))
											})

											Context("Composer API Service not found", func() {
												BeforeEach(func() {
													serviceRepository.EXPECT().Read(requestContext, composerComposerAPIServiceName, operatorNamespace).Return(nil, errNotFound)
												})

												It("Should return an error if failed to create the composer api service", func() {
													// given
													serviceRepository.EXPECT().Create(requestContext, &composerComposerAPIService).Return(errFailed)
													// when
													result, err := reconciler.Reconcile(requestContext, request)
													// then
													Expect(err).To(Equal(errFailed))
													Expect(result).To(Equal(resultRequeue))
												})

												It("Should return requeue if the composer api service was created", func() {
													// given
													serviceRepository.EXPECT().Create(requestContext, &composerComposerAPIService).Return(nil)
													// when
													result, err := reconciler.Reconcile(requestContext, request)
													// then
													Expect(err).To(BeNil())
													Expect(result).To(Equal(resultQuickRequeue))
												})
											})

											Context("Composer API Service exists", func() {
												const (
													composerComposerAPIPortName = "composer-api"
													composerWorkerAPIPortName   = "worker-api"
												)

												var (
													composerWorkerAPIService corev1.Service

													workerAPIExternalPort = intstr.FromInt(8700)
												)
												BeforeEach(func() {
													serviceRepository.EXPECT().Read(requestContext, composerComposerAPIServiceName, operatorNamespace).Return(&composerComposerAPIService, nil)

													composerWorkerAPIService = corev1.Service{
														ObjectMeta: metav1.ObjectMeta{
															Namespace:       operatorNamespace,
															Name:            composerWorkerAPIServiceName,
															OwnerReferences: []metav1.OwnerReference{ownerReference},
														},
														Spec: corev1.ServiceSpec{
															Type: corev1.ServiceTypeClusterIP,
															Ports: []corev1.ServicePort{
																{
																	Name:       composerWorkerAPIPortName,
																	Port:       443,
																	Protocol:   "TCP",
																	TargetPort: workerAPIExternalPort,
																},
															},
															Selector: map[string]string{
																"app": "osbuild-composer",
															},
														},
													}
												})

												It("Should return an error if failed to get the worker api service", func() {
													// given
													serviceRepository.EXPECT().Read(requestContext, composerWorkerAPIServiceName, operatorNamespace).Return(nil, errFailed)
													// when
													result, err := reconciler.Reconcile(requestContext, request)
													// then
													Expect(err).To(Equal(errFailed))
													Expect(result).To(Equal(resultRequeue))
												})

												Context("Worker API Service not found", func() {
													BeforeEach(func() {
														serviceRepository.EXPECT().Read(requestContext, composerWorkerAPIServiceName, operatorNamespace).Return(nil, errNotFound)
													})

													It("Should return an error if failed to create the worker api service", func() {
														// given
														serviceRepository.EXPECT().Create(requestContext, &composerWorkerAPIService).Return(errFailed)
														// when
														result, err := reconciler.Reconcile(requestContext, request)
														// then
														Expect(err).To(Equal(errFailed))
														Expect(result).To(Equal(resultRequeue))
													})

													It("Should return requeue if the worker api service was created", func() {
														// given
														serviceRepository.EXPECT().Create(requestContext, &composerWorkerAPIService).Return(nil)
														// when
														result, err := reconciler.Reconcile(requestContext, request)
														// then
														Expect(err).To(BeNil())
														Expect(result).To(Equal(resultQuickRequeue))
													})
												})

												Context("Worker API Service exists", func() {
													const (
														workerConfigAnsibleConfigConfigMapName = "osbuild-worker-setup-ansible-config"
													)

													BeforeEach(func() {
														serviceRepository.EXPECT().Read(requestContext, composerWorkerAPIServiceName, operatorNamespace).Return(&composerWorkerAPIService, nil)
													})

													It("Should return an error if failed to get the configMap for the ansible config ", func() {
														// given
														configMapRepository.EXPECT().Read(requestContext, workerConfigAnsibleConfigConfigMapName, operatorNamespace).Return(nil, errFailed)
														// when
														result, err := reconciler.Reconcile(requestContext, request)
														// then
														Expect(err).To(Equal(errFailed))
														Expect(result).To(Equal(resultRequeue))
													})

													Context("ConfigMap for configuration ansible config not found", func() {
														BeforeEach(func() {
															configMapRepository.EXPECT().Read(requestContext, workerConfigAnsibleConfigConfigMapName, operatorNamespace).Return(nil, errNotFound)
														})

														It("Should return an error if failed to create the configMap for the ansible config for the worker configuration job", func() {
															// given
															configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
															// when
															result, err := reconciler.Reconcile(requestContext, request)
															// then
															Expect(err).To(Equal(errFailed))
															Expect(result).To(Equal(resultRequeue))
														})

														It("Should return requeue if succeeded to create the configMap for the ansible config for the worker configuration job", func() {
															// given
															configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
															// when
															result, err := reconciler.Reconcile(requestContext, request)
															// then
															Expect(err).To(BeNil())
															Expect(result).To(Equal(resultQuickRequeue))
														})
													})

													Context("ConfigMap for the ansible config exists", func() {
														const (
															workerOSBuildWorkerConfigConfigMapName = "osbuild-worker-config"
														)

														var (
															workerConfigAnsibleConfigConfigMap = corev1.ConfigMap{
																ObjectMeta: metav1.ObjectMeta{
																	Namespace: operatorNamespace,
																	Name:      workerConfigAnsibleConfigConfigMapName,
																},
															}
														)

														BeforeEach(func() {
															configMapRepository.EXPECT().Read(requestContext, workerConfigAnsibleConfigConfigMapName, operatorNamespace).Return(&workerConfigAnsibleConfigConfigMap, nil)
														})

														It("Should return an error if failed to get the configMap for the osbuild-worker config", func() {
															// given
															configMapRepository.EXPECT().Read(requestContext, workerOSBuildWorkerConfigConfigMapName, operatorNamespace).Return(nil, errFailed)
															// when
															result, err := reconciler.Reconcile(requestContext, request)
															// then
															Expect(err).To(Equal(errFailed))
															Expect(result).To(Equal(resultRequeue))
														})

														Context("ConfigMap for osbuild-worker config not found", func() {
															BeforeEach(func() {
																configMapRepository.EXPECT().Read(requestContext, workerOSBuildWorkerConfigConfigMapName, operatorNamespace).Return(nil, errNotFound)
															})

															It("Should return an error if failed to create the configMap for the osbuild-worker config", func() {
																// given
																configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
																// when
																result, err := reconciler.Reconcile(requestContext, request)
																// then
																Expect(err).To(Equal(errFailed))
																Expect(result).To(Equal(resultRequeue))
															})

															It("Should return requeue if succeeded to create the configMap for the osbuild-worker config", func() {
																// given
																configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
																// when
																result, err := reconciler.Reconcile(requestContext, request)
																// then
																Expect(err).To(BeNil())
																Expect(result).To(Equal(resultQuickRequeue))
															})
														})

														Context("ConfigMap for the osbuild-worker config exists", func() {
															const (
																workerSSHKeysSecretName    = "osbuild-worker-ssh" // #nosec G101
																workerSSHKeysPrivateKeyKey = "ssh-privatekey"
																workerSSHKeysPublicKeyKey  = "ssh-publickey"
															)

															var (
																workerOSBuildWorkerConfigConfigMap = corev1.ConfigMap{
																	ObjectMeta: metav1.ObjectMeta{
																		Namespace: operatorNamespace,
																		Name:      workerOSBuildWorkerConfigConfigMapName,
																	},
																}

																sshPrivateKey = []byte("sshPrivateKey")
																sshPublicKey  = []byte("sshPublicKey")

																workerSSHKeysSecret *corev1.Secret
															)

															BeforeEach(func() {
																configMapRepository.EXPECT().Read(requestContext, workerOSBuildWorkerConfigConfigMapName, operatorNamespace).Return(&workerOSBuildWorkerConfigConfigMap, nil)
																workerSSHKeysSecret = &corev1.Secret{
																	ObjectMeta: metav1.ObjectMeta{
																		Name:            workerSSHKeysSecretName,
																		Namespace:       operatorNamespace,
																		OwnerReferences: []metav1.OwnerReference{ownerReference},
																	},
																	Immutable: pointer.Bool(true),
																	StringData: map[string]string{
																		workerSSHKeysPrivateKeyKey: string(sshPrivateKey),
																		workerSSHKeysPublicKeyKey:  string(sshPublicKey),
																	},
																}
															})

															It("Should return an error if failed to get the secret for the osbuild-worker ssh keys", func() {
																// given
																secretRepository.EXPECT().Read(requestContext, workerSSHKeysSecretName, operatorNamespace).Return(nil, errFailed)
																// when
																result, err := reconciler.Reconcile(requestContext, request)
																// then
																Expect(err).To(Equal(errFailed))
																Expect(result).To(Equal(resultRequeue))
															})

															Context("Secret for osbuild-worker ssh keys not found", func() {
																BeforeEach(func() {
																	secretRepository.EXPECT().Read(requestContext, workerSSHKeysSecretName, operatorNamespace).Return(nil, errNotFound)
																})

																It("Should return en error if failed to create the ssh keys", func() {
																	// given
																	sshGenerateError := fmt.Errorf("failed to generate ssh keys")
																	sshKeyGenerator.EXPECT().GenerateSSHKeyPair().Return(nil, nil, sshGenerateError)
																	// when
																	result, err := reconciler.Reconcile(requestContext, request)
																	// then
																	Expect(err).To(Equal(sshGenerateError))
																	Expect(result).To(Equal(resultRequeue))
																})

																Context("Successfully created the ssh keys", func() {
																	BeforeEach(func() {
																		sshKeyGenerator.EXPECT().GenerateSSHKeyPair().Return(sshPrivateKey, sshPublicKey, nil)
																	})

																	It("Should return an error if failed to create the secret for the osbuild-worker ssh keys", func() {
																		// given
																		secretRepository.EXPECT().Create(requestContext, workerSSHKeysSecret).Return(errFailed)
																		// when
																		result, err := reconciler.Reconcile(requestContext, request)
																		// then
																		Expect(err).To(Equal(errFailed))
																		Expect(result).To(Equal(resultRequeue))
																	})

																	It("Should return requeue if succeeded to create the secret for the osbuild-worker ssh keys", func() {
																		// given
																		secretRepository.EXPECT().Create(requestContext, workerSSHKeysSecret).Return(nil)
																		// when
																		result, err := reconciler.Reconcile(requestContext, request)
																		// then
																		Expect(err).To(BeNil())
																		Expect(result).To(Equal(resultQuickRequeue))
																	})
																})
															})

															Context("Secret for osbuild-worker ssh keys exists", func() {
																BeforeEach(func() {
																	secretRepository.EXPECT().Read(requestContext, workerSSHKeysSecretName, operatorNamespace).Return(workerSSHKeysSecret, nil)
																})

																It("Should return an error if failed to get the VirtualMachine for the internal builder", func() {
																	// given
																	virtualMachineRepository.EXPECT().Read(requestContext, internalBuilderName, operatorNamespace).Return(nil, errFailed)
																	// when
																	result, err := reconciler.Reconcile(requestContext, request)
																	// then
																	Expect(err).To(Equal(errFailed))
																	Expect(result).To(Equal(resultRequeue))
																})

																Context("VirtualMachine for the internal builder not found", func() {
																	BeforeEach(func() {
																		virtualMachineRepository.EXPECT().Read(requestContext, internalBuilderName, operatorNamespace).Return(nil, errNotFound)
																	})

																	It("Should return an error if failed to create the VirtualMachine for the internal builder", func() {
																		// given
																		virtualMachineRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
																		// when
																		result, err := reconciler.Reconcile(requestContext, request)
																		// then
																		Expect(err).To(Equal(errFailed))
																		Expect(result).To(Equal(resultRequeue))
																	})

																	It("Should return requeue if succeeded to create the VirtualMachine for the internal builder", func() {
																		// given
																		virtualMachineRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
																		// when
																		result, err := reconciler.Reconcile(requestContext, request)
																		// then
																		Expect(err).To(BeNil())
																		Expect(result).To(Equal(resultQuickRequeue))
																	})
																})

																Context("VirtualMachine for the internal builder exists", func() {
																	const (
																		workerSSHServiceNameFormat = "worker-%s-ssh"
																		workerSSHPortName          = "ssh"
																	)

																	var (
																		workerSSHServiceName = fmt.Sprintf(workerSSHServiceNameFormat, internalBuilderName)
																		internalBuilderVM    *kubevirtv1.VirtualMachine
																		workerSSHService     *corev1.Service
																	)

																	BeforeEach(func() {
																		internalBuilderVM = &kubevirtv1.VirtualMachine{
																			ObjectMeta: metav1.ObjectMeta{
																				Name:      internalBuilderName,
																				Namespace: operatorNamespace,
																			},
																		}
																		virtualMachineRepository.EXPECT().Read(requestContext, internalBuilderName, operatorNamespace).Return(internalBuilderVM, nil)

																		workerSSHService = &corev1.Service{
																			ObjectMeta: metav1.ObjectMeta{
																				Namespace:       operatorNamespace,
																				Name:            workerSSHServiceName,
																				OwnerReferences: []metav1.OwnerReference{ownerReference},
																			},
																			Spec: corev1.ServiceSpec{
																				Type: corev1.ServiceTypeClusterIP,
																				Ports: []corev1.ServicePort{
																					{
																						Name:       workerSSHPortName,
																						Port:       22,
																						Protocol:   "TCP",
																						TargetPort: intstr.FromInt(22),
																					},
																				},
																				Selector: map[string]string{
																					"osbuild-worker": internalBuilderName,
																				},
																			},
																		}
																	})

																	It("Should return an error if failed to get the ssh service of the internal builder", func() {
																		// given
																		serviceRepository.EXPECT().Read(requestContext, workerSSHServiceName, operatorNamespace).Return(nil, errFailed)
																		// when
																		result, err := reconciler.Reconcile(requestContext, request)
																		// then
																		Expect(err).To(Equal(errFailed))
																		Expect(result).To(Equal(resultRequeue))
																	})

																	Context("SSH Service for the internal builder not found", func() {
																		BeforeEach(func() {
																			serviceRepository.EXPECT().Read(requestContext, workerSSHServiceName, operatorNamespace).Return(nil, errNotFound)
																		})

																		It("Should return an error if failed to create the SSH Service for the internal builder", func() {
																			// given
																			serviceRepository.EXPECT().Create(requestContext, workerSSHService).Return(errFailed)
																			// when
																			result, err := reconciler.Reconcile(requestContext, request)
																			// then
																			Expect(err).To(Equal(errFailed))
																			Expect(result).To(Equal(resultRequeue))
																		})

																		It("Should return requeue if succeeded to create the SSH Service for the internal builder", func() {
																			// given
																			serviceRepository.EXPECT().Create(requestContext, workerSSHService).Return(nil)
																			// when
																			result, err := reconciler.Reconcile(requestContext, request)
																			// then
																			Expect(err).To(BeNil())
																			Expect(result).To(Equal(resultQuickRequeue))
																		})
																	})

																	Context("SSH Service for the internal builder exists", func() {
																		BeforeEach(func() {
																			serviceRepository.EXPECT().Read(requestContext, workerSSHServiceName, operatorNamespace).Return(workerSSHService, nil)
																		})

																		It("Should fail if failed to get the VM of the internal worker", func() {
																			// given
																			virtualMachineRepository.EXPECT().Read(requestContext, internalBuilderName, operatorNamespace).Return(nil, errFailed)
																			// when
																			result, err := reconciler.Reconcile(requestContext, request)
																			// then
																			Expect(err).To(Equal(errFailed))
																			Expect(result).To(Equal(resultRequeue))
																		})

																		It("Should return requeue if the internal worker VM is not ready", func() {
																			// given
																			virtualMachineRepository.EXPECT().Read(requestContext, internalBuilderName, operatorNamespace).Return(internalBuilderVM, nil)
																			// when
																			result, err := reconciler.Reconcile(requestContext, request)
																			// then
																			Expect(err).To(BeNil())
																			Expect(result).To(Equal(resultQuickRequeue))
																		})

																		Context("Internal worker VM is ready", func() {
																			const (
																				workerCertificateNameFormat = "worker-%s-cert"
																			)

																			var (
																				internalBuilderCertificateName = fmt.Sprintf(workerCertificateNameFormat, internalBuilderName)
																				internalBuilderCertificate     *certmanagerv1.Certificate
																			)

																			BeforeEach(func() {
																				internalBuilderVM.Status.Conditions = []kubevirtv1.VirtualMachineCondition{
																					{
																						Type:   kubevirtv1.VirtualMachineReady,
																						Status: corev1.ConditionTrue,
																					},
																				}
																				virtualMachineRepository.EXPECT().Read(requestContext, internalBuilderName, operatorNamespace).Return(internalBuilderVM, nil)
																				internalBuilderCertificate = &certmanagerv1.Certificate{
																					ObjectMeta: metav1.ObjectMeta{
																						Name:            internalBuilderCertificateName,
																						Namespace:       operatorNamespace,
																						OwnerReferences: []metav1.OwnerReference{ownerReference},
																					},
																					Spec: certmanagerv1.CertificateSpec{
																						SecretName: internalBuilderCertificateName,
																						PrivateKey: &certmanagerv1.CertificatePrivateKey{
																							Algorithm: "ECDSA",
																							Size:      256,
																						},
																						DNSNames: []string{
																							internalBuilderName,
																						},
																						Duration: &metav1.Duration{
																							Duration: time.Hour * certificateDuration,
																						},
																						IssuerRef: certmanagermetav1.ObjectReference{
																							Group: "cert-manager.io",
																							Kind:  "Issuer",
																							Name:  caIssuerName,
																						},
																					},
																				}
																			})

																			It("Should return an error if failed to get the certificate of the internal builder", func() {
																				// given
																				certificateRepository.EXPECT().Read(requestContext, internalBuilderCertificateName, operatorNamespace).Return(nil, errFailed)
																				// when
																				result, err := reconciler.Reconcile(requestContext, request)
																				// then
																				Expect(err).To(Equal(errFailed))
																				Expect(result).To(Equal(resultRequeue))
																			})

																			Context("Certificate for the internal builder not found", func() {
																				BeforeEach(func() {
																					certificateRepository.EXPECT().Read(requestContext, internalBuilderCertificateName, operatorNamespace).Return(nil, errNotFound)
																				})

																				It("Should return an error if failed to create the Certificate for the internal builder", func() {
																					// given
																					certificateRepository.EXPECT().Create(requestContext, internalBuilderCertificate).Return(errFailed)
																					// when
																					result, err := reconciler.Reconcile(requestContext, request)
																					// then
																					Expect(err).To(Equal(errFailed))
																					Expect(result).To(Equal(resultRequeue))
																				})

																				It("Should return requeue if succeeded to create the Certificate for the internal builder", func() {
																					// given
																					certificateRepository.EXPECT().Create(requestContext, internalBuilderCertificate).Return(nil)
																					// when
																					result, err := reconciler.Reconcile(requestContext, request)
																					// then
																					Expect(err).To(BeNil())
																					Expect(result).To(Equal(resultQuickRequeue))
																				})
																			})

																			Context("Certificate for the internal builder exists", func() {
																				var (
																					internalBuilderCertificateSecret *corev1.Secret
																				)
																				BeforeEach(func() {
																					certificateRepository.EXPECT().Read(requestContext, internalBuilderCertificateName, operatorNamespace).Return(internalBuilderCertificate, nil)
																					internalBuilderCertificateSecret = &corev1.Secret{
																						ObjectMeta: metav1.ObjectMeta{
																							Namespace: operatorNamespace,
																							Name:      internalBuilderCertificateName,
																						},
																					}
																				})

																				It("Should return an error if failed to get the secret of the certificate of the internal builder", func() {
																					// given
																					secretRepository.EXPECT().Read(requestContext, internalBuilderCertificateName, operatorNamespace).Return(nil, errFailed)
																					// when
																					result, err := reconciler.Reconcile(requestContext, request)
																					// then
																					Expect(err).To(Equal(errFailed))
																					Expect(result).To(Equal(resultRequeue))
																				})

																				Context("The Secret of the Certificate for the internal builder is not owned by the instance", func() {
																					var (
																						originalInternalBuilderCertificateSecret *corev1.Secret
																						updatedInternalBuilderCertificateSecret  *corev1.Secret
																					)

																					BeforeEach(func() {
																						secretRepository.EXPECT().Read(requestContext, internalBuilderCertificateName, operatorNamespace).Return(internalBuilderCertificateSecret, nil)

																						originalInternalBuilderCertificateSecret = internalBuilderCertificateSecret.DeepCopy()
																						updatedInternalBuilderCertificateSecret = internalBuilderCertificateSecret.DeepCopy()
																						updatedInternalBuilderCertificateSecret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerReference}
																					})

																					It("Should return an error if failed to own the secret of the Certificate for the internal builder", func() {
																						// given
																						secretRepository.EXPECT().Patch(requestContext, originalInternalBuilderCertificateSecret, updatedInternalBuilderCertificateSecret).Return(errFailed)
																						// when
																						result, err := reconciler.Reconcile(requestContext, request)
																						// then
																						Expect(err).To(Equal(errFailed))
																						Expect(result).To(Equal(resultRequeue))
																					})

																					It("Should return requeue if succeeded to set the owner of the secret of the Certificate for the internal builder", func() {
																						// given
																						secretRepository.EXPECT().Patch(requestContext, originalInternalBuilderCertificateSecret, updatedInternalBuilderCertificateSecret).Return(nil)
																						// when
																						result, err := reconciler.Reconcile(requestContext, request)
																						// then
																						Expect(err).To(BeNil())
																						Expect(result).To(Equal(resultQuickRequeue))
																					})
																				})

																				Context("Internal Builder Certificate Secret Owner is already set", func() {
																					const (
																						workerSetupPlaybookConfigMapNameFormat = "worker-%s-setup-playbook"
																					)
																					var (
																						internalBuilderSetupPlaybookConfigMapName = fmt.Sprintf(workerSetupPlaybookConfigMapNameFormat, internalBuilderName)
																					)
																					BeforeEach(func() {
																						internalBuilderCertificateSecret.ObjectMeta.OwnerReferences = []metav1.OwnerReference{ownerReference}
																						secretRepository.EXPECT().Read(requestContext, internalBuilderCertificateName, operatorNamespace).Return(internalBuilderCertificateSecret, nil)
																					})

																					It("Should return an error if failed to get the configMap for the setup playbook of the internal builder", func() {
																						// given
																						configMapRepository.EXPECT().Read(requestContext, internalBuilderSetupPlaybookConfigMapName, operatorNamespace).Return(nil, errFailed)
																						// when
																						result, err := reconciler.Reconcile(requestContext, request)
																						// then
																						Expect(err).To(Equal(errFailed))
																						Expect(result).To(Equal(resultRequeue))
																					})

																					Context("The configMap for the setup playbook for the internal builder not found", func() {
																						BeforeEach(func() {
																							configMapRepository.EXPECT().Read(requestContext, internalBuilderSetupPlaybookConfigMapName, operatorNamespace).Return(nil, errNotFound)
																						})

																						It("Should return an error if failed to create the configMap for the setup playbook for the internal builder", func() {
																							// given
																							configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
																							// when
																							result, err := reconciler.Reconcile(requestContext, request)
																							// then
																							Expect(err).To(Equal(errFailed))
																							Expect(result).To(Equal(resultRequeue))
																						})

																						It("Should return requeue if succeeded to create the configMap for the setup playbook for the internal builder", func() {
																							// given
																							configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
																							// when
																							result, err := reconciler.Reconcile(requestContext, request)
																							// then
																							Expect(err).To(BeNil())
																							Expect(result).To(Equal(resultQuickRequeue))
																						})
																					})

																					Context("The configMap for the setup playbook for the internal builder exists", func() {
																						const (
																							workerSetupInventoryConfigMapNameFormat = "worker-%s-inventory"
																						)
																						var (
																							internalBuilderSetupInventoryConfigMapName = fmt.Sprintf(workerSetupInventoryConfigMapNameFormat, internalBuilderName)
																							internalBuilderSetupPlaybookConfigMap      = &corev1.ConfigMap{
																								ObjectMeta: metav1.ObjectMeta{
																									Name:      internalBuilderSetupPlaybookConfigMapName,
																									Namespace: operatorNamespace,
																								},
																							}
																						)

																						BeforeEach(func() {
																							configMapRepository.EXPECT().Read(requestContext, internalBuilderSetupPlaybookConfigMapName, operatorNamespace).Return(internalBuilderSetupPlaybookConfigMap, nil)
																						})

																						It("Should return an error if failed to get the configMap for the setup inventory of the internal builder", func() {
																							// given
																							configMapRepository.EXPECT().Read(requestContext, internalBuilderSetupInventoryConfigMapName, operatorNamespace).Return(nil, errFailed)
																							// when
																							result, err := reconciler.Reconcile(requestContext, request)
																							// then
																							Expect(err).To(Equal(errFailed))
																							Expect(result).To(Equal(resultRequeue))
																						})

																						Context("The configMap for the setup playbook for the internal builder not found", func() {
																							BeforeEach(func() {
																								configMapRepository.EXPECT().Read(requestContext, internalBuilderSetupInventoryConfigMapName, operatorNamespace).Return(nil, errNotFound)
																							})

																							It("Should return an error if failed to create the configMap for the setup inventory for the internal builder", func() {
																								// given
																								configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
																								// when
																								result, err := reconciler.Reconcile(requestContext, request)
																								// then
																								Expect(err).To(Equal(errFailed))
																								Expect(result).To(Equal(resultRequeue))
																							})

																							It("Should return requeue if succeeded to create the configMap for the setup inventory for the internal builder", func() {
																								// given
																								configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
																								// when
																								result, err := reconciler.Reconcile(requestContext, request)
																								// then
																								Expect(err).To(BeNil())
																								Expect(result).To(Equal(resultQuickRequeue))
																							})
																						})

																						Context("The configMap for the setup inventory for the internal builder exists", func() {
																							const (
																								workerSetupJobNameFormat = "worker-%s-setup"
																							)
																							var (
																								internalBuilderSetupJobName            = fmt.Sprintf(workerSetupJobNameFormat, internalBuilderName)
																								internalBuilderSetupInventoryConfigMap = &corev1.ConfigMap{
																									ObjectMeta: metav1.ObjectMeta{
																										Name:      internalBuilderSetupInventoryConfigMapName,
																										Namespace: operatorNamespace,
																									},
																								}
																							)

																							BeforeEach(func() {
																								configMapRepository.EXPECT().Read(requestContext, internalBuilderSetupInventoryConfigMapName, operatorNamespace).Return(internalBuilderSetupInventoryConfigMap, nil)
																							})

																							It("Should return an error if failed to get the job for the setup of the internal builder", func() {
																								// given
																								jobRepository.EXPECT().Read(requestContext, internalBuilderSetupJobName, operatorNamespace).Return(nil, errFailed)
																								// when
																								result, err := reconciler.Reconcile(requestContext, request)
																								// then
																								Expect(err).To(Equal(errFailed))
																								Expect(result).To(Equal(resultRequeue))
																							})

																							Context("The job for the setup for the internal builder not found", func() {
																								BeforeEach(func() {
																									jobRepository.EXPECT().Read(requestContext, internalBuilderSetupJobName, operatorNamespace).Return(nil, errNotFound)
																								})

																								It("Should return an error if failed to create the job for the setup for the internal builder", func() {
																									// given
																									jobRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
																									// when
																									result, err := reconciler.Reconcile(requestContext, request)
																									// then
																									Expect(err).To(Equal(errFailed))
																									Expect(result).To(Equal(resultRequeue))
																								})

																								It("Should return requeue if succeeded to create the job for the setup for the internal builder", func() {
																									// given
																									jobRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
																									// when
																									result, err := reconciler.Reconcile(requestContext, request)
																									// then
																									Expect(err).To(BeNil())
																									Expect(result).To(Equal(resultQuickRequeue))
																								})
																							})

																							Context("The job for the setup for the internal builder exists", func() {
																								var (
																									internalBuilderSetupJob = &batchv1.Job{
																										ObjectMeta: metav1.ObjectMeta{
																											Name:      internalBuilderSetupJobName,
																											Namespace: operatorNamespace,
																										},
																									}
																								)

																								BeforeEach(func() {
																									jobRepository.EXPECT().Read(requestContext, internalBuilderSetupJobName, operatorNamespace).Return(internalBuilderSetupJob, nil)
																								})

																								Context("The certificate for the external builder exists", func() {
																									var (
																										externalBuilderCertificateName = fmt.Sprintf(workerCertificateNameFormat, externalBuilderName)
																										externalBuilderCertificate     *certmanagerv1.Certificate
																									)

																									BeforeEach(func() {
																										externalBuilderCertificate = &certmanagerv1.Certificate{
																											ObjectMeta: metav1.ObjectMeta{
																												Name:            externalBuilderCertificateName,
																												Namespace:       operatorNamespace,
																												OwnerReferences: []metav1.OwnerReference{ownerReference},
																											},
																											Spec: certmanagerv1.CertificateSpec{
																												SecretName: externalBuilderCertificateName,
																												PrivateKey: &certmanagerv1.CertificatePrivateKey{
																													Algorithm: "ECDSA",
																													Size:      256,
																												},
																												DNSNames: []string{
																													externalBuilderName,
																												},
																												Duration: &metav1.Duration{
																													Duration: time.Hour * certificateDuration,
																												},
																												IssuerRef: certmanagermetav1.ObjectReference{
																													Group: "cert-manager.io",
																													Kind:  "Issuer",
																													Name:  caIssuerName,
																												},
																											},
																										}
																										certificateRepository.EXPECT().Read(requestContext, externalBuilderCertificateName, operatorNamespace).Return(externalBuilderCertificate, nil)
																									})

																									Context("The Secret of the certificate of the external worker is owned by the instance", func() {
																										var (
																											externalBuilderCertificateSecret *corev1.Secret
																										)

																										BeforeEach(func() {
																											externalBuilderCertificateSecret = &corev1.Secret{
																												ObjectMeta: metav1.ObjectMeta{
																													Namespace:       operatorNamespace,
																													Name:            internalBuilderCertificateName,
																													OwnerReferences: []metav1.OwnerReference{ownerReference},
																												},
																											}
																											secretRepository.EXPECT().Read(requestContext, externalBuilderCertificateName, operatorNamespace).Return(externalBuilderCertificateSecret, nil)
																										})

																										Context("The ConfigMap for the playbook of the external builder setup exists", func() {
																											var (
																												externalBuilderSetupPlaybookConfigMapName = fmt.Sprintf(workerSetupPlaybookConfigMapNameFormat, externalBuilderName)
																												externalBuilderSetupPlaybookConfigMap     = &corev1.ConfigMap{
																													ObjectMeta: metav1.ObjectMeta{
																														Name:      externalBuilderSetupPlaybookConfigMapName,
																														Namespace: operatorNamespace,
																													},
																												}
																											)

																											BeforeEach(func() {
																												configMapRepository.EXPECT().Read(requestContext, externalBuilderSetupPlaybookConfigMapName, operatorNamespace).Return(externalBuilderSetupPlaybookConfigMap, nil)
																											})

																											Context("The ConfigMap for the inventory of the external builder setup exists", func() {
																												var (
																													externalBuilderSetupInventoryConfigMapName = fmt.Sprintf(workerSetupInventoryConfigMapNameFormat, externalBuilderName)
																													externalBuilderSetupInventoryConfigMap     = &corev1.ConfigMap{
																														ObjectMeta: metav1.ObjectMeta{
																															Name:      externalBuilderSetupInventoryConfigMapName,
																															Namespace: operatorNamespace,
																														},
																													}
																												)

																												BeforeEach(func() {
																													configMapRepository.EXPECT().Read(requestContext, externalBuilderSetupInventoryConfigMapName, operatorNamespace).Return(externalBuilderSetupInventoryConfigMap, nil)
																												})

																												Context("The job for the external builder setup exists", func() {
																													var (
																														externalBuilderSetupJobName = fmt.Sprintf(workerSetupJobNameFormat, externalBuilderName)
																														externalBuilderSetupJob     = &batchv1.Job{
																															ObjectMeta: metav1.ObjectMeta{
																																Name:      externalBuilderSetupJobName,
																																Namespace: operatorNamespace,
																															},
																														}
																													)

																													BeforeEach(func() {
																														jobRepository.EXPECT().Read(requestContext, externalBuilderSetupJobName, operatorNamespace).Return(externalBuilderSetupJob, nil)
																													})

																													It("Should return Done", func() {
																														// when
																														result, err := reconciler.Reconcile(requestContext, request)
																														// then
																														Expect(err).To(BeNil())
																														Expect(result).To(Equal(resultDone))
																													})

																												})

																											})

																										})

																									})
																								})
																							})
																						})
																					})
																				})
																			})
																		})
																	})
																})
															})
														})
													})
												})
											})
										})
									})
								})
							})
						})
					})
				})
			})
		})
	})
})
