package webhook_test

import (
	"bytes"
	"encoding/json"
	"io"
	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stestclient "k8s.io/client-go/kubernetes/fake"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/1password/kubernetes-secrets-injector/pkg/webhook"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func createRequest(body io.Reader) *http.Request {
	req, err := http.NewRequest("POST", "/inject", body)
	Expect(err).NotTo(HaveOccurred())

	req.Header.Set("Content-Type", "application/json")

	return req
}

func parseResponseBody(rr *httptest.ResponseRecorder) *admissionv1.AdmissionResponse {
	var resBody *admissionv1.AdmissionReview
	err2 := json.Unmarshal(rr.Body.Bytes(), &resBody)
	Expect(err2).NotTo(HaveOccurred())

	return resBody.Response
}

var _ = Describe("Webhook Test", Ordered, func() {
	var rr *httptest.ResponseRecorder
	var handler http.HandlerFunc

	BeforeAll(func() {
		webhook.K8sClient = &webhook.Client{
			Clientset: k8stestclient.NewSimpleClientset(),
		}
		secretInjector := webhook.SecretInjector{}
		handler = secretInjector.Serve
	})

	BeforeEach(func() {
		rr = httptest.NewRecorder()
	})

	AfterEach(func() {
		rr = nil
	})

	It("should return bad request if no body provided", func() {
		req := createRequest(nil)
		handler.ServeHTTP(rr, req)

		Expect(rr.Code).To(Equal(http.StatusBadRequest))
	})

	It("should return StatusUnsupportedMediaType if Content-Type is not application/json", func() {
		req := createRequest(strings.NewReader("some content"))
		req.Header.Del("Content-Type")
		handler.ServeHTTP(rr, req)

		Expect(rr.Code).To(Equal(http.StatusUnsupportedMediaType))
	})

	It("should return error if no pod provided", func() {
		ar := admissionv1.AdmissionReview{
			Request: &admissionv1.AdmissionRequest{
				Namespace: "default",
			},
		}
		body, err := json.Marshal(ar)
		Expect(err).NotTo(HaveOccurred())

		req := createRequest(bytes.NewReader(body))
		handler.ServeHTTP(rr, req)

		Expect(rr.Code).To(Equal(http.StatusOK))

		responseBody := parseResponseBody(rr)
		Expect(responseBody.Result.Message).To(Equal("unexpected end of JSON input"))
	})

	Context("NOT create a patch", func() {
		When("pod has no inject annotation", func() {
			It("should NOT create patch", func() {
				pod := corev1.Pod{}
				raw, err := json.Marshal(pod)
				Expect(err).NotTo(HaveOccurred())

				ar := admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Namespace: "default",
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}
				body, err := json.Marshal(ar)
				Expect(err).NotTo(HaveOccurred())

				req := createRequest(bytes.NewReader(body))
				handler.ServeHTTP(rr, req)

				responseBody := parseResponseBody(rr)
				Expect(responseBody.Patch).To(BeNil())
			})
		})

		When("inject annotation is empty", func() {
			It("should NOT create patch", func() {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"operator.1password.io/inject": "",
						},
					},
				}
				raw, err := json.Marshal(pod)
				Expect(err).NotTo(HaveOccurred())

				ar := admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Namespace: "default",
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}
				body, err := json.Marshal(ar)
				Expect(err).NotTo(HaveOccurred())

				req := createRequest(bytes.NewReader(body))
				handler.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK))

				responseBody := parseResponseBody(rr)
				Expect(responseBody.Patch).To(BeNil())
			})
		})

		When("inject status annotation is 'injected'", func() {
			It("should NOT create patch", func() {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"operator.1password.io/status": "injected",
						},
					},
				}
				raw, err := json.Marshal(pod)
				Expect(err).NotTo(HaveOccurred())

				ar := admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Namespace: "default",
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}
				body, err := json.Marshal(ar)
				Expect(err).NotTo(HaveOccurred())

				req := createRequest(bytes.NewReader(body))
				handler.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK))

				responseBody := parseResponseBody(rr)
				Expect(responseBody.Patch).To(BeNil())
			})
		})

		When("inject label does not have a match with container name", func() {
			It("should NOT create patch", func() {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"operator.1password.io/inject": "not-existing-container",
						},
					},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{Name: "app-init"},
						},
						Containers: []corev1.Container{
							{Name: "app"},
						},
					},
				}
				raw, err := json.Marshal(pod)
				Expect(err).NotTo(HaveOccurred())

				ar := admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Namespace: "default",
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}
				body, err := json.Marshal(ar)
				Expect(err).NotTo(HaveOccurred())

				req := createRequest(bytes.NewReader(body))
				handler.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK))

				responseBody := parseResponseBody(rr)
				Expect(responseBody.Patch).To(BeNil())
			})
		})

		When("container does not have a command provided", func() {
			It("should NOT create patch", func() {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"operator.1password.io/inject": "app",
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{Name: "app"},
						},
					},
				}
				raw, err := json.Marshal(pod)
				Expect(err).NotTo(HaveOccurred())

				ar := admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Namespace: "default",
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}
				body, err := json.Marshal(ar)
				Expect(err).NotTo(HaveOccurred())

				req := createRequest(bytes.NewReader(body))
				handler.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK))

				responseBody := parseResponseBody(rr)
				Expect(responseBody.Patch).To(BeNil())
			})
		})

		When("init-container does not have a command provided", func() {
			It("should NOT create patch", func() {
				pod := corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Annotations: map[string]string{
							"operator.1password.io/inject": "app",
						},
					},
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{Name: "app"},
						},
					},
				}
				raw, err := json.Marshal(pod)
				Expect(err).NotTo(HaveOccurred())

				ar := admissionv1.AdmissionReview{
					Request: &admissionv1.AdmissionRequest{
						Namespace: "default",
						Object: runtime.RawExtension{
							Raw: raw,
						},
					},
				}
				body, err := json.Marshal(ar)
				Expect(err).NotTo(HaveOccurred())

				req := createRequest(bytes.NewReader(body))
				handler.ServeHTTP(rr, req)

				Expect(rr.Code).To(Equal(http.StatusOK))

				responseBody := parseResponseBody(rr)
				Expect(responseBody.Patch).To(BeNil())
			})
		})
	})

	// TODO: cover cases when patch is created, check it contains env vars, volumes etc.
})
