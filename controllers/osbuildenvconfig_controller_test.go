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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"

	certmanagerv1 "github.com/cert-manager/cert-manager/pkg/apis/certmanager/v1"
	certmanagermetav1 "github.com/cert-manager/cert-manager/pkg/apis/meta/v1"
	osbuildv1alpha1 "github.com/project-flotta/osbuild-operator/api/v1alpha1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	. "github.com/onsi/gomega"

	"github.com/project-flotta/osbuild-operator/internal/conf"
	"github.com/project-flotta/osbuild-operator/internal/repository/certificate"
	"github.com/project-flotta/osbuild-operator/internal/repository/configmap"
	"github.com/project-flotta/osbuild-operator/internal/repository/deployment"
	"github.com/project-flotta/osbuild-operator/internal/repository/osbuildenvconfig"
	"github.com/project-flotta/osbuild-operator/internal/repository/secret"
	"github.com/project-flotta/osbuild-operator/internal/repository/service"

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
	)

	var (
		mockCtrl *gomock.Controller

		scheme *runtime.Scheme

		osBuildEnvConfigRepository *osbuildenvconfig.MockRepository
		certificateRepository      *certificate.MockRepository
		configMapRepository        *configmap.MockRepository
		deploymentRepository       *deployment.MockRepository
		serviceRepository          *service.MockRepository
		secretRepository           *secret.MockRepository

		reconciler     *controllers.OSBuildEnvConfigReconciler
		requestContext context.Context

		errNotFound error
		errFailed   error

		instance osbuildv1alpha1.OSBuildEnvConfig

		request = ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name: instanceName,
			},
		}

		resultRequeue = ctrl.Result{Requeue: true}
		resultDone    = ctrl.Result{}

		composerCertificateSecret = corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: operatorNamespace,
				Name:      composerCertificateName,
			},
		}
	)

	BeforeEach(func() {
		os.Setenv("WORKING_NAMESPACE", operatorNamespace)
		os.Setenv("CA_ISSUER_NAME", caIssuerName)
		os.Setenv("TEMPLATES_DIR", templatesDir)
		err := conf.Load()
		Expect(err).To(BeNil())

		mockCtrl = gomock.NewController(GinkgoT())

		osBuildEnvConfigRepository = osbuildenvconfig.NewMockRepository(mockCtrl)
		certificateRepository = certificate.NewMockRepository(mockCtrl)
		configMapRepository = configmap.NewMockRepository(mockCtrl)
		deploymentRepository = deployment.NewMockRepository(mockCtrl)
		serviceRepository = service.NewMockRepository(mockCtrl)
		secretRepository = secret.NewMockRepository(mockCtrl)

		scheme = runtime.NewScheme()
		err = clientgoscheme.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = osbuildv1alpha1.AddToScheme(scheme)
		Expect(err).To(BeNil())
		err = certmanagerv1.AddToScheme(scheme)
		Expect(err).To(BeNil())

		reconciler = &controllers.OSBuildEnvConfigReconciler{
			Scheme:                     scheme,
			OSBuildEnvConfigRepository: osBuildEnvConfigRepository,
			CertificateRepository:      certificateRepository,
			ConfigMapRepository:        configMapRepository,
			DeploymentRepository:       deploymentRepository,
			ServiceRepository:          serviceRepository,
			SecretRepository:           secretRepository,
		}

		requestContext = context.TODO()

		errNotFound = errors.NewNotFound(schema.GroupResource{}, "Requested resource was not found")
		errFailed = errors.NewInternalError(fmt.Errorf("Server encounter and error"))

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
			},
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
			BeforeEach(func() {
				instance.ObjectMeta.Finalizers = []string{osBuildOperatorFinalizer}
			})

			AfterEach(func() {
				instance.ObjectMeta.Finalizers = nil
			})

			It("Should return requeue error if failed to read the secret", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
				secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errFailed)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(Equal(errFailed))
				Expect(result).To(Equal(resultRequeue))
			})

			It("Should return error if failed to delete the secret", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
				secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(&composerCertificateSecret, nil)
				secretRepository.EXPECT().Delete(requestContext, &composerCertificateSecret).Return(errFailed)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(Equal(errFailed))
				Expect(result).To(Equal(resultRequeue))
			})

			It("Should return requeue if deleted the secret", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
				secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(&composerCertificateSecret, nil)
				secretRepository.EXPECT().Delete(requestContext, &composerCertificateSecret).Return(nil)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(BeNil())
				Expect(result).To(Equal(resultRequeue))
			})

			It("Should return error if failed to remove finalizer", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
				secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errNotFound)
				originalInstance := instance.DeepCopy()
				osBuildEnvConfigRepository.EXPECT().Patch(requestContext, originalInstance, &instance).Return(errFailed)
				// when
				result, err := reconciler.Reconcile(requestContext, request)
				// then
				Expect(err).To(Equal(errFailed))
				Expect(result).To(Equal(resultRequeue))
			})

			It("Should return Done if removed the finalizer successfully", func() {
				// given
				osBuildEnvConfigRepository.EXPECT().Read(requestContext, instanceName).Return(&instance, nil)
				secretRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(nil, errNotFound)
				originalInstance := instance.DeepCopy()
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
				Expect(result).To(Equal(resultRequeue))
			})
		})

		Context("With finalizer", func() {
			const (
				certificateDuration            = 87600
				composerComposerAPIServiceName = "osbuild-composer"
				composerWorkerAPIServiceName   = "osbuild-worker"
			)
			var (
				composerCertificate certmanagerv1.Certificate
			)
			BeforeEach(func() {
				instance.ObjectMeta.Finalizers = []string{osBuildOperatorFinalizer}

				composerCertificate = certmanagerv1.Certificate{
					ObjectMeta: metav1.ObjectMeta{
						Name:      composerCertificateName,
						Namespace: operatorNamespace,
						OwnerReferences: []metav1.OwnerReference{
							{
								APIVersion:         instance.APIVersion,
								Kind:               instance.Kind,
								Name:               instance.Name,
								UID:                instance.UID,
								BlockOwnerDeletion: pointer.BoolPtr(true),
								Controller:         pointer.BoolPtr(true),
							},
						},
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
				Expect(result).To(Equal(resultRequeue))
			})

			Context("Certificate is already created", func() {
				BeforeEach(func() {
					certificateRepository.EXPECT().Read(requestContext, composerCertificateName, operatorNamespace).Return(&composerCertificate, nil)
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

					It("Should return an error if failed to get the configmap for the osbuild-composer configuration", func() {
						// given
						configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(nil, errFailed)
						// when
						result, err := reconciler.Reconcile(requestContext, request)
						// then
						Expect(err).To(Equal(errFailed))
						Expect(result).To(Equal(resultRequeue))
					})

					It("Should return an error if failed to create the configmap for the osbuild-composer configuration", func() {
						// given
						configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(nil, errNotFound)
						configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
						// when
						result, err := reconciler.Reconcile(requestContext, request)
						// then
						Expect(err).To(Equal(errFailed))
						Expect(result).To(Equal(resultRequeue))
					})

					It("Should return requeue if the configmap for the osbuild-composer configuration was created", func() {
						// given
						configMapRepository.EXPECT().Read(requestContext, composerConfigMapName, operatorNamespace).Return(nil, errNotFound)
						configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
						// when
						result, err := reconciler.Reconcile(requestContext, request)
						// then
						Expect(err).To(BeNil())
						Expect(result).To(Equal(resultRequeue))
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

						It("Should return an error if failed to get the configmap for the proxy configuration", func() {
							// given
							configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(nil, errFailed)
							// when
							result, err := reconciler.Reconcile(requestContext, request)
							// then
							Expect(err).To(Equal(errFailed))
							Expect(result).To(Equal(resultRequeue))
						})

						It("Should return an error if failed to create the configmap for the proxy configuration", func() {
							// given
							configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(nil, errNotFound)
							configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(errFailed)
							// when
							result, err := reconciler.Reconcile(requestContext, request)
							// then
							Expect(err).To(Equal(errFailed))
							Expect(result).To(Equal(resultRequeue))
						})

						It("Should return requeue if the configmap for the proxy configuration was created", func() {
							// given
							configMapRepository.EXPECT().Read(requestContext, composerProxyConfigMapName, operatorNamespace).Return(nil, errNotFound)
							configMapRepository.EXPECT().Create(requestContext, gomock.Any()).Return(nil)
							// when
							result, err := reconciler.Reconcile(requestContext, request)
							// then
							Expect(err).To(BeNil())
							Expect(result).To(Equal(resultRequeue))
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
									Expect(result).To(Equal(resultRequeue))
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
											Namespace: operatorNamespace,
											Name:      composerComposerAPIServiceName,
											OwnerReferences: []metav1.OwnerReference{
												{
													APIVersion:         instance.APIVersion,
													Kind:               instance.Kind,
													Name:               instance.Name,
													UID:                instance.UID,
													BlockOwnerDeletion: pointer.BoolPtr(true),
													Controller:         pointer.BoolPtr(true),
												},
											},
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
										Expect(result).To(Equal(resultRequeue))
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
												Namespace: operatorNamespace,
												Name:      composerWorkerAPIServiceName,
												OwnerReferences: []metav1.OwnerReference{
													{
														APIVersion:         instance.APIVersion,
														Kind:               instance.Kind,
														Name:               instance.Name,
														UID:                instance.UID,
														BlockOwnerDeletion: pointer.BoolPtr(true),
														Controller:         pointer.BoolPtr(true),
													},
												},
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
											Expect(result).To(Equal(resultRequeue))
										})
									})

									Context("Composer API Service exists", func() {
										BeforeEach(func() {
											serviceRepository.EXPECT().Read(requestContext, composerWorkerAPIServiceName, operatorNamespace).Return(&composerWorkerAPIService, nil)
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
