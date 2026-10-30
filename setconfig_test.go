package main

import (
	"strings"
	"testing"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestSetConfigHidesKeyParseError(t *testing.T) {
	const leak = "SECRET-FRAGMENT-1f3c"
	client := fake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sa-key", Namespace: "cert-manager"},
		Data:       map[string][]byte{"key.json": []byte(`{"id":"x","` + leak + `":"` + leak + `"`)},
	})
	s := &yandexCloudDNSSolver{client: client}
	err := s.setConfig(&v1alpha1.ChallengeRequest{
		ResourceNamespace: "cert-manager",
		Config:            &extapi.JSON{Raw: []byte(`{"folder":"f","serviceAccountSecretRef":{"name":"sa-key","key":"key.json"}}`)},
	})
	if err == nil {
		t.Fatal("setConfig returned nil error for an invalid key")
	}
	const want = `failed to parse service account key from secret "cert-manager/sa-key"`
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
	if strings.Contains(err.Error(), leak) {
		t.Errorf("error leaks key content: %q", err.Error())
	}
}
