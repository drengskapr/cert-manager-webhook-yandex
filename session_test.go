package main

import (
	"context"
	"strings"
	"testing"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/apis/acme/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	extapi "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const keyLeak = "SECRET-FRAGMENT-1f3c"

// invalidKeyChallenge returns a solver whose Secret holds an unparsable key, and a ChallengeRequest using it.
func invalidKeyChallenge() (*yandexCloudDNSSolver, *v1alpha1.ChallengeRequest) {
	client := fake.NewClientset(&corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sa-key", Namespace: "cert-manager"},
		Data:       map[string][]byte{"key.json": []byte(`{"id":"x","` + keyLeak + `":"` + keyLeak + `"`)},
	})
	return &yandexCloudDNSSolver{client: client}, &v1alpha1.ChallengeRequest{
		ResourceNamespace: "cert-manager",
		ResolvedZone:      "example.com.",
		ResolvedFQDN:      "_acme-challenge.example.com.",
		Config:            &extapi.JSON{Raw: []byte(`{"folder":"f","serviceAccountSecretRef":{"name":"sa-key","key":"key.json"}}`)},
	}
}

const wantKeyParseError = `failed to parse service account key from secret "cert-manager/sa-key"`

func TestNewDNSSessionHidesKeyParseError(t *testing.T) {
	s, ch := invalidKeyChallenge()
	_, err := s.newDNSSession(context.Background(), ch)
	if err == nil {
		t.Fatal("newDNSSession returned nil error for an invalid key")
	}
	if err.Error() != wantKeyParseError {
		t.Errorf("error = %q, want %q", err.Error(), wantKeyParseError)
	}
	if strings.Contains(err.Error(), keyLeak) {
		t.Errorf("error leaks key content: %q", err.Error())
	}
}

// A fresh solver, as after a pod restart between Present and CleanUp.
func TestCleanUpLoadsConfig(t *testing.T) {
	s, ch := invalidKeyChallenge()
	err := s.CleanUp(ch)
	if err == nil || err.Error() != wantKeyParseError {
		t.Fatalf("CleanUp error = %v, want %q", err, wantKeyParseError)
	}
}
