package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	jsonpatch "github.com/evanphx/json-patch/v5"
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

func sendPodAndGetResponse(pod corev1.Pod, rr *httptest.ResponseRecorder, handler http.HandlerFunc) *admissionv1.AdmissionResponse {
	raw, err := json.Marshal(pod)
	Expect(err).NotTo(HaveOccurred())
	ar := admissionv1.AdmissionReview{
		Request: &admissionv1.AdmissionRequest{
			Namespace: "default",
			Object:    runtime.RawExtension{Raw: raw},
		},
	}
	body, err := json.Marshal(ar)
	Expect(err).NotTo(HaveOccurred())
	req := createRequest(bytes.NewReader(body))
	handler.ServeHTTP(rr, req)
	return parseResponseBody(rr)
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

	Context("preserves existing pod template annotations", func() {
		It("does not overwrite annotations; adds only status via per-key patch", func() {
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.1password.io/inject":  "app",
						"operator.1password.io/version": "2-beta",
						"myannotation":                  "mine",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Command: []string{"sleep", "infinity"}},
					},
				},
			}
			responseBody := sendPodAndGetResponse(pod, rr, handler)
			Expect(responseBody.Patch).NotTo(BeNil())

			var patch []patchOperation
			Expect(json.Unmarshal(responseBody.Patch, &patch)).To(Succeed())
			// Must not replace the whole annotations object (which would drop myannotation).
			var hasOverwriteOp bool
			for _, op := range patch {
				if op.Path == "/metadata/annotations" && op.Op == "add" {
					hasOverwriteOp = true
					break
				}
			}
			Expect(hasOverwriteOp).To(BeFalse(),
				"patch must not add whole /metadata/annotations when pod already has annotations")

			// Status must be set via per-key path so we don't overwrite other annotations.
			var foundStatusPatch bool
			for _, op := range patch {
				if op.Path == "/metadata/annotations/operator.1password.io~1status" && (op.Op == "add" || op.Op == "replace") {
					foundStatusPatch = true
					Expect(op.Value).To(Equal("injected"))
					break
				}
			}
			Expect(foundStatusPatch).To(BeTrue(), "patch should set operator.1password.io/status via per-key path")
		})

		It("applying the patch preserves custom annotations and sets status", func() {
			pod := corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						"operator.1password.io/inject":  "app",
						"operator.1password.io/version": "2-beta",
						"myannotation":                  "mine",
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{Name: "app", Command: []string{"sleep", "infinity"}},
					},
				},
			}
			raw, err := json.Marshal(pod)
			Expect(err).NotTo(HaveOccurred())

			responseBody := sendPodAndGetResponse(pod, rr, handler)
			Expect(responseBody.Patch).NotTo(BeNil())

			patch, err := jsonpatch.DecodePatch(responseBody.Patch)
			Expect(err).NotTo(HaveOccurred())
			patchedRaw, err := patch.Apply(raw)
			Expect(err).NotTo(HaveOccurred())

			var patched corev1.Pod
			Expect(json.Unmarshal(patchedRaw, &patched)).To(Succeed())

			Expect(patched.Annotations).To(HaveKeyWithValue("operator.1password.io/status", "injected"))
			Expect(patched.Annotations).To(HaveKeyWithValue("myannotation", "mine"))
		})
	})

})
