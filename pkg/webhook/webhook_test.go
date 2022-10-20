package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stestclient "k8s.io/client-go/kubernetes/fake"
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

var testNotPatch = map[string]struct {
	pod corev1.Pod
}{
	"Pod has no inject annotation": {
		pod: corev1.Pod{},
	},
	"Inject annotation is empty": {
		pod: corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"operator.1password.io/inject": "",
				},
			},
		},
	},
	"Inject status annotation is 'injected'": {
		pod: corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"operator.1password.io/status": "injected",
				},
			},
		},
	},
	"Inject label does not have a match with container name": {
		pod: corev1.Pod{
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
		},
	},
	"Container does not have a command provided": {
		pod: corev1.Pod{
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
		},
	},
	"Init-container does not have a command provided": {
		pod: corev1.Pod{
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
		},
	},
}

var _ = Describe("Webhook Test", Ordered, func() {
	var rr *httptest.ResponseRecorder
	var handler http.HandlerFunc

	BeforeAll(func() {
		k8sClient = k8stestclient.NewSimpleClientset()
		secretInjector := SecretInjector{}
		handler = secretInjector.Serve
	})

	BeforeEach(func() {
		rr = httptest.NewRecorder()
	})

	AfterEach(func() {
		rr = nil
	})

	It("Should return bad request if no body provided", func() {
		req := createRequest(nil)
		handler.ServeHTTP(rr, req)

		Expect(rr.Code).To(Equal(http.StatusBadRequest))
	})

	It("Should return StatusUnsupportedMediaType if Content-Type is not application/json", func() {
		req := createRequest(strings.NewReader("some content"))
		req.Header.Del("Content-Type")
		handler.ServeHTTP(rr, req)

		Expect(rr.Code).To(Equal(http.StatusUnsupportedMediaType))
	})

	It("Should return error if no pod provided", func() {
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
		for testCase, testData := range testNotPatch {
			When(testCase, func() {
				It("Should NOT create patch", func() {
					raw, err := json.Marshal(testData.pod)
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
		}
	})

	// TODO: cover cases when patch is created, check it contains env vars, volumes etc.
})
