package webhook

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/1password/kubernetes-secrets-injector/pkg/utils"
	"github.com/1password/kubernetes-secrets-injector/version"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
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

var testPatch = map[string]struct {
	pod         corev1.Pod
	expectPatch []map[string]interface{}
}{
	"Pod without user-defined init containers has inject annotation that matches container name and command is defined": {
		pod: corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"operator.1password.io/inject": "app",
				},
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{{Name: "app", Command: []string{"echo", "hello"}}},
			},
		},
		expectPatch: []map[string]interface{}{
			{
				"op":   "add",
				"path": "/spec/initContainers",
				"value": []map[string]interface{}{
					{
						"name":      "copy-op-bin",
						"image":     "1password/op:2",
						"command":   []string{"sh", "-c", "cp /usr/local/bin/op /op/bin/"},
						"resources": map[string]interface{}{},
						"volumeMounts": []map[string]interface{}{
							{
								"name":      "op-bin",
								"mountPath": "/op/bin/",
							},
						},
						"imagePullPolicy": "IfNotPresent",
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/volumeMounts",
				"value": []map[string]interface{}{
					{
						"mountPath": "/op/bin/",
						"name":      "op-bin",
						"readOnly":  true,
					},
				},
			},
			{
				"op":   "replace",
				"path": "/spec/containers/0/command",
				"value": []string{
					"/op/bin/op", "run", "--", "echo", "hello",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env",
				"value": []map[string]interface{}{
					{
						"name":  "OP_INTEGRATION_NAME",
						"value": "1Password Kubernetes Webhook",
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_ID",
					"value": "K8W",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_BUILDNUMBER",
					"value": utils.MakeBuildVersion(version.Version),
				},
			},
			{
				"op":   "add",
				"path": "/spec/volumes",
				"value": []map[string]interface{}{
					{
						"name": "op-bin",
						"emptyDir": map[string]string{
							"medium": "Memory",
						},
					},
				},
			},
			{
				"op":   "add",
				"path": "/metadata/annotations",
				"value": map[string]string{
					"operator.1password.io/status": "injected",
				},
			},
		},
	},
	"Pod with user-defined init container and operator.1password.io/injector-init-first annotation not set": {
		pod: corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"operator.1password.io/inject": "app,init-app",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "init-app", Command: []string{"echo", "hello"}}},
				Containers:     []corev1.Container{{Name: "app", Command: []string{"echo", "hello"}}},
			},
		},
		expectPatch: []map[string]interface{}{
			{
				"op":   "add",
				"path": "/spec/initContainers/-",
				"value": map[string]interface{}{
					"name":      "copy-op-bin",
					"image":     "1password/op:2",
					"command":   []string{"sh", "-c", "cp /usr/local/bin/op /op/bin/"},
					"resources": map[string]interface{}{},
					"volumeMounts": []map[string]interface{}{
						{
							"name":      "op-bin",
							"mountPath": "/op/bin/",
						},
					},
					"imagePullPolicy": "IfNotPresent",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/volumeMounts",
				"value": []map[string]interface{}{
					{
						"mountPath": "/op/bin/",
						"name":      "op-bin",
						"readOnly":  true,
					},
				},
			},
			{
				"op":   "replace",
				"path": "/spec/containers/0/command",
				"value": []string{
					"/op/bin/op", "run", "--", "echo", "hello",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env",
				"value": []map[string]interface{}{
					{
						"name":  "OP_INTEGRATION_NAME",
						"value": "1Password Kubernetes Webhook",
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_ID",
					"value": "K8W",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_BUILDNUMBER",
					"value": utils.MakeBuildVersion(version.Version),
				},
			},
			{
				"op":   "add",
				"path": "/spec/volumes",
				"value": []map[string]interface{}{
					{
						"name": "op-bin",
						"emptyDir": map[string]string{
							"medium": "Memory",
						},
					},
				},
			},
			{
				"op":   "add",
				"path": "/metadata/annotations",
				"value": map[string]string{
					"operator.1password.io/status": "injected",
				},
			},
		},
	},
	"Pod with 2 containers, a user-defined init container and operator.1password.io/injector-init-first annotation set to true": {
		pod: corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{
				Annotations: map[string]string{
					"operator.1password.io/inject":              "app,app2,init-app",
					"operator.1password.io/injector-init-first": "true",
				},
			},
			Spec: corev1.PodSpec{
				InitContainers: []corev1.Container{{Name: "init-app", Command: []string{"echo", "hello"}}},
				Containers: []corev1.Container{
					{Name: "app", Command: []string{"echo", "hello"}},
					{Name: "app2", Command: []string{"echo", "hello"}},
				},
			},
		},
		expectPatch: []map[string]interface{}{
			{
				"op":   "add",
				"path": "/spec/initContainers/0/volumeMounts",
				"value": []map[string]interface{}{
					{
						"mountPath": "/op/bin/",
						"name":      "op-bin",
						"readOnly":  true,
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/initContainers/-",
				"value": map[string]interface{}{
					"command":   []string{"echo", "hello"},
					"name":      "init-app",
					"resources": map[string]interface{}{},
				},
			},
			{
				"op":    "replace",
				"path":  "/spec/initContainers/0/command",
				"value": []string{"/op/bin/op", "run", "--", "echo", "hello"},
			},
			{
				"op":   "add",
				"path": "/spec/initContainers/0/env",
				"value": []map[string]interface{}{
					{
						"name":  "OP_INTEGRATION_NAME",
						"value": "1Password Kubernetes Webhook",
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/initContainers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_ID",
					"value": "K8W",
				},
			},
			{
				"op":   "add",
				"path": "/spec/initContainers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_BUILDNUMBER",
					"value": utils.MakeBuildVersion(version.Version),
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/volumeMounts",
				"value": []map[string]interface{}{
					{
						"mountPath": "/op/bin/",
						"name":      "op-bin",
						"readOnly":  true,
					},
				},
			},
			{
				"op":    "replace",
				"path":  "/spec/containers/0/command",
				"value": []string{"/op/bin/op", "run", "--", "echo", "hello"},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env",
				"value": []map[string]interface{}{
					{
						"name":  "OP_INTEGRATION_NAME",
						"value": "1Password Kubernetes Webhook",
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_ID",
					"value": "K8W",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/0/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_BUILDNUMBER",
					"value": utils.MakeBuildVersion(version.Version),
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/1/volumeMounts",
				"value": []map[string]interface{}{
					{
						"mountPath": "/op/bin/",
						"name":      "op-bin",
						"readOnly":  true,
					},
				},
			},
			{
				"op":    "replace",
				"path":  "/spec/containers/1/command",
				"value": []string{"/op/bin/op", "run", "--", "echo", "hello"},
			},
			{
				"op":   "add",
				"path": "/spec/containers/1/env",
				"value": []map[string]interface{}{
					{
						"name":  "OP_INTEGRATION_NAME",
						"value": "1Password Kubernetes Webhook",
					},
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/1/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_ID",
					"value": "K8W",
				},
			},
			{
				"op":   "add",
				"path": "/spec/containers/1/env/-",
				"value": map[string]string{
					"name":  "OP_INTEGRATION_BUILDNUMBER",
					"value": utils.MakeBuildVersion(version.Version),
				},
			},
			{
				"op":   "add",
				"path": "/spec/volumes",
				"value": []map[string]interface{}{
					{
						"name": "op-bin",
						"emptyDir": map[string]string{
							"medium": "Memory",
						},
					},
				},
			},
			{
				"op":   "add",
				"path": "/metadata/annotations",
				"value": map[string]string{
					"operator.1password.io/status": "injected",
				},
			},
			{
				"op":   "add",
				"path": "/spec/initContainers/-",
				"value": map[string]interface{}{
					"name":      "copy-op-bin",
					"image":     "1password/op:2",
					"command":   []string{"sh", "-c", "cp /usr/local/bin/op /op/bin/"},
					"resources": map[string]interface{}{},
					"volumeMounts": []map[string]interface{}{
						{
							"name":      "op-bin",
							"mountPath": "/op/bin/",
						},
					},
					"imagePullPolicy": "IfNotPresent",
				},
			},
			{
				"op":   "add",
				"path": "/spec/initContainers/0/volumeMounts",
				"value": []map[string]interface{}{
					{
						"mountPath": "/op/bin/",
						"name":      "op-bin",
						"readOnly":  true,
					},
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
					pod := testData.pod.DeepCopy()
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
		}
	})

	Context("CREATE a patch", func() {
		for testCase, testData := range testPatch {
			tc := testCase
			td := testData
			pod := td.pod.DeepCopy()

			When(tc, func() {
				It("Should correctly patch pod", func() {
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
					Expect(responseBody.Allowed).To(BeTrue())
					Expect(responseBody.Patch).NotTo(BeNil())

					var patchOps []map[string]interface{}
					Expect(json.Unmarshal(responseBody.Patch, &patchOps)).To(Succeed())

					Expect(len(td.expectPatch)).To(Equal(len(patchOps)))
					for _, expectedOp := range td.expectPatch {
						found := false
						for _, actualOp := range patchOps {
							if cmp.Equal(
								normalize(expectedOp),
								normalize(actualOp),
								cmpopts.SortMaps(func(a, b string) bool { return a < b }),
							) {
								found = true
								break
							}
						}

						Expect(found).To(BeTrue(),
							"Did not find expected patch operation in the produced patchOps:\nExpected:\n%v\n\nProduced:\n%v",
							expectedOp, patchOps,
						)
					}
				})
			})
		}
	})
})

func normalize(obj interface{}) interface{} {
	bytes, err := json.Marshal(obj)
	Expect(err).NotTo(HaveOccurred())

	var normalized interface{}
	Expect(json.Unmarshal(bytes, &normalized)).To(Succeed())
	return normalized
}
