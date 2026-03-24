package controllers

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	helxv1 "github.com/helxplatform/helxapp-controller/api/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("HelxApp Controller", func() {

	Context("When creating CRDs", func() {

		// Test 66: TestCreateHelxApp
		It("should create a HelxApp successfully", func() {
			ctx := context.Background()

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns-app"}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			app := &helxv1.HelxApp{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "helx.renci.org/v1",
					Kind:       "HelxApp",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-app",
					Namespace: "test-ns-app",
				},
				Spec: helxv1.HelxAppSpec{
					AppClassName: "TestApp",
					Services: []helxv1.Service{
						{
							Name:  "web",
							Image: "nginx:latest",
							Ports: []helxv1.PortMap{
								{ContainerPort: 80, Port: 8080},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).Should(Succeed())

			retrieved := &helxv1.HelxApp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-app",
				Namespace: "test-ns-app",
			}, retrieved)).Should(Succeed())
			Expect(retrieved.Spec.AppClassName).To(Equal("TestApp"))
			Expect(retrieved.Spec.Services).To(HaveLen(1))
			Expect(retrieved.Spec.Services[0].Name).To(Equal("web"))
		})

		// Test 67: TestCreateHelxUser
		It("should create a HelxUser successfully", func() {
			ctx := context.Background()

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns-user"}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			userHandle := "jdoe"
			user := &helxv1.HelxUser{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "helx.renci.org/v1",
					Kind:       "HelxUser",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-user",
					Namespace: "test-ns-user",
				},
				Spec: helxv1.HelxUserSpec{
					UserHandle: &userHandle,
				},
			}
			Expect(k8sClient.Create(ctx, user)).Should(Succeed())

			retrieved := &helxv1.HelxUser{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-user",
				Namespace: "test-ns-user",
			}, retrieved)).Should(Succeed())
			Expect(retrieved.Spec.UserHandle).NotTo(BeNil())
			Expect(*retrieved.Spec.UserHandle).To(Equal("jdoe"))
		})

		// Test 68: TestCreateHelxInst
		It("should create a HelxInst successfully", func() {
			ctx := context.Background()

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns-inst"}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			inst := &helxv1.HelxInst{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "helx.renci.org/v1",
					Kind:       "HelxInst",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-inst",
					Namespace: "test-ns-inst",
				},
				Spec: helxv1.HelxInstSpec{
					AppName:  "test-app",
					UserName: "jdoe",
				},
			}
			Expect(k8sClient.Create(ctx, inst)).Should(Succeed())

			retrieved := &helxv1.HelxInst{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "test-inst",
				Namespace: "test-ns-inst",
			}, retrieved)).Should(Succeed())
			Expect(retrieved.Spec.AppName).To(Equal("test-app"))
			Expect(retrieved.Spec.UserName).To(Equal("jdoe"))
		})

		// Test 69: TestCRDFieldValidation
		It("should preserve all fields on a fully populated HelxApp", func() {
			ctx := context.Background()

			ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: "test-ns-validation"}}
			Expect(k8sClient.Create(ctx, ns)).Should(Succeed())

			runAsUser := int64(1000)
			runAsGroup := int64(1000)
			fsGroup := int64(2000)

			app := &helxv1.HelxApp{
				TypeMeta: metav1.TypeMeta{
					APIVersion: "helx.renci.org/v1",
					Kind:       "HelxApp",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "full-app",
					Namespace: "test-ns-validation",
				},
				Spec: helxv1.HelxAppSpec{
					AppClassName: "FullApp",
					SourceText:   "source-content",
					Services: []helxv1.Service{
						{
							Name:    "api-server",
							Image:   "myregistry/api:v2",
							Command: []string{"/bin/sh", "-c", "run-server"},
							Environment: map[string]string{
								"ENV_VAR_ONE": "value1",
								"ENV_VAR_TWO": "value2",
							},
							Init: false,
							Ports: []helxv1.PortMap{
								{ContainerPort: 8080, Port: 80},
								{ContainerPort: 8443, Port: 443},
							},
							ResourceBounds: map[string]helxv1.ResourceBoundary{
								"cpu": {
									Min: "100m",
									Max: "2",
								},
								"memory": {
									Min: "128Mi",
									Max: "1Gi",
								},
							},
							SecurityContext: &helxv1.SecurityContext{
								RunAsUser:          &runAsUser,
								RunAsGroup:         &runAsGroup,
								FSGroup:            &fsGroup,
								SupplementalGroups: []int64{3000, 4000},
							},
							Volumes: map[string]string{
								"/data":  "data-pvc",
								"/cache": "cache-pvc",
							},
						},
						{
							Name:  "sidecar",
							Image: "busybox:latest",
							Init:  true,
							Ports: []helxv1.PortMap{
								{ContainerPort: 9090},
							},
						},
					},
				},
			}
			Expect(k8sClient.Create(ctx, app)).Should(Succeed())

			retrieved := &helxv1.HelxApp{}
			Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      "full-app",
				Namespace: "test-ns-validation",
			}, retrieved)).Should(Succeed())

			// Verify top-level spec fields
			Expect(retrieved.Spec.AppClassName).To(Equal("FullApp"))
			Expect(retrieved.Spec.SourceText).To(Equal("source-content"))
			Expect(retrieved.Spec.Services).To(HaveLen(2))

			// Verify first service in detail
			svc := retrieved.Spec.Services[0]
			Expect(svc.Name).To(Equal("api-server"))
			Expect(svc.Image).To(Equal("myregistry/api:v2"))
			Expect(svc.Command).To(Equal([]string{"/bin/sh", "-c", "run-server"}))
			Expect(svc.Init).To(BeFalse())

			// Environment
			Expect(svc.Environment).To(HaveLen(2))
			Expect(svc.Environment["ENV_VAR_ONE"]).To(Equal("value1"))
			Expect(svc.Environment["ENV_VAR_TWO"]).To(Equal("value2"))

			// Ports
			Expect(svc.Ports).To(HaveLen(2))
			Expect(svc.Ports[0].ContainerPort).To(Equal(int32(8080)))
			Expect(svc.Ports[0].Port).To(Equal(int32(80)))
			Expect(svc.Ports[1].ContainerPort).To(Equal(int32(8443)))
			Expect(svc.Ports[1].Port).To(Equal(int32(443)))

			// ResourceBounds
			Expect(svc.ResourceBounds).To(HaveLen(2))
			Expect(svc.ResourceBounds["cpu"].Min).To(Equal("100m"))
			Expect(svc.ResourceBounds["cpu"].Max).To(Equal("2"))
			Expect(svc.ResourceBounds["memory"].Min).To(Equal("128Mi"))
			Expect(svc.ResourceBounds["memory"].Max).To(Equal("1Gi"))

			// SecurityContext
			Expect(svc.SecurityContext).NotTo(BeNil())
			Expect(*svc.SecurityContext.RunAsUser).To(Equal(int64(1000)))
			Expect(*svc.SecurityContext.RunAsGroup).To(Equal(int64(1000)))
			Expect(*svc.SecurityContext.FSGroup).To(Equal(int64(2000)))
			Expect(svc.SecurityContext.SupplementalGroups).To(Equal([]int64{3000, 4000}))

			// Volumes
			Expect(svc.Volumes).To(HaveLen(2))
			Expect(svc.Volumes["/data"]).To(Equal("data-pvc"))
			Expect(svc.Volumes["/cache"]).To(Equal("cache-pvc"))

			// Verify second service
			svc2 := retrieved.Spec.Services[1]
			Expect(svc2.Name).To(Equal("sidecar"))
			Expect(svc2.Image).To(Equal("busybox:latest"))
			Expect(svc2.Init).To(BeTrue())
			Expect(svc2.Ports).To(HaveLen(1))
			Expect(svc2.Ports[0].ContainerPort).To(Equal(int32(9090)))
		})
	})
})
