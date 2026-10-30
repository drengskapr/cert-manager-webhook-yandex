#!/usr/bin/env bash
# Asserts the rendered chart manifests. Requires helm and yq v4 (mikefarah).
set -euo pipefail

cd "$(dirname "$0")/.."
CHART=deploy/cert-manager-webhook-yandex
TMP=$(mktemp -d)
trap 'rm -rf "$TMP"' EXIT

failures=0
case_name=""

render() {
	local release=$1 ns=$2
	shift 2
	helm template "$release" "$CHART" -n "$ns" "$@" >"$TMP/$case_name.yaml"
}

# q prints scalars as plain lines; qj prints compact JSON. A yq error yields a value no check expects.
q() { yq e -N "$1" "$TMP/$case_name.yaml" || echo "<yq error>"; }
qj() { yq e -N -o=json -I=0 "$1" "$TMP/$case_name.yaml" || echo "<yq error>"; }

check() {
	local desc=$1 actual=$2 expected=$3
	if [[ "$actual" != "$expected" ]]; then
		printf 'FAIL [%s] %s\n  got:  %s\n  want: %s\n' "$case_name" "$desc" "$actual" "$expected"
		failures=$((failures + 1))
	fi
}

DEP='select(.kind == "Deployment") | .spec.template.spec'
CTR="$DEP | .containers[0]"
SVC='select(.kind == "Service") | .spec'
API='select(.kind == "APIService")'
SERVING='select(.kind == "Certificate" and .spec.isCA != true)'

subjects() { q "select(.kind == \"$1\" and .metadata.name == \"$2\") | .subjects[] | .namespace + \"/\" + .name"; }

# The Deployment's TLS Secret, the serving Certificate's secretName and the APIService CA source must agree.
check_tls() {
	local ns=$1 tls
	tls=$(q "$DEP | .volumes[] | select(.name == \"certs\") | .secret.secretName")
	check "serving Certificate secretName" "$(q "$SERVING | .spec.secretName")" "$tls"
	check "APIService inject-ca-from" "$(q "$API | .metadata.annotations.\"cert-manager.io/inject-ca-from\"")" "$ns/$tls"
}

if ! lint=$(helm lint "$CHART" 2>&1); then
	printf 'FAIL [lint]\n%s\n' "$lint"
	failures=$((failures + 1))
else
	echo "$lint" | tail -n 1
fi

case_name=default
F=yandex-cert-manager-webhook-yandex
render yandex cert-manager
check "kind counts" "$(q '.kind' | sort | uniq -c | awk '{print $2 "=" $1}' | paste -sd, -)" \
	"APIService=1,Certificate=2,ClusterRole=2,ClusterRoleBinding=3,Deployment=1,Issuer=2,RoleBinding=1,Service=1,ServiceAccount=1"
check "Deployment name" "$(q 'select(.kind == "Deployment") | .metadata.name')" "$F"
check "replicas" "$(q 'select(.kind == "Deployment") | .spec.replicas')" "1"
check "image" "$(q "$CTR | .image")" "ghcr.io/drengskapr/cert-manager-webhook-yandex:0.1.0"
check "imagePullPolicy" "$(q "$CTR | .imagePullPolicy")" "IfNotPresent"
check "no imagePullSecrets" "$(q "$DEP | has(\"imagePullSecrets\")")" "false"
check "args" "$(qj "$CTR | .args")" \
	'["--tls-cert-file=/tls/tls.crt","--tls-private-key-file=/tls/tls.key","--secure-port=6443","--logging-format=json-rfc3339"]'
check "containerPort" "$(q "$CTR | .ports[0].containerPort")" "6443"
check "env names" "$(qj "$CTR | .env | map(.name)")" '["GROUP_NAME"]'
check "GROUP_NAME" "$(q "$CTR | .env[] | select(.name == \"GROUP_NAME\") | .value")" "acme.cloud.yandex.com"
check "pod runAsNonRoot" "$(q "$DEP | .securityContext.runAsNonRoot")" "true"
check "pod runAsUser" "$(q "$DEP | .securityContext.runAsUser")" "65532"
check "pod runAsGroup" "$(q "$DEP | .securityContext.runAsGroup")" "65532"
check "pod seccompProfile" "$(q "$DEP | .securityContext.seccompProfile.type")" "RuntimeDefault"
check "allowPrivilegeEscalation" "$(q "$CTR | .securityContext.allowPrivilegeEscalation")" "false"
check "readOnlyRootFilesystem" "$(q "$CTR | .securityContext.readOnlyRootFilesystem")" "true"
check "capabilities.drop" "$(qj "$CTR | .securityContext.capabilities.drop")" '["ALL"]'
check "Service port" "$(q "$SVC | .ports[0].port")" "443"
check "Service port tag" "$(q "$SVC | .ports[0].port | tag")" '!!int'
check "Service targetPort" "$(q "$SVC | .ports[0].targetPort")" "https"
check "APIService name" "$(q "$API | .metadata.name")" "v1alpha1.acme.cloud.yandex.com"
check "APIService service.port" "$(q "$API | .spec.service.port")" "443"
check "groupPriorityMinimum" "$(q "$API | .spec.groupPriorityMinimum")" "1000"
check "versionPriority" "$(q "$API | .spec.versionPriority")" "15"
check "TLS volume secretName" "$(q "$DEP | .volumes[] | select(.name == \"certs\") | .secret.secretName")" "$F-tls"
check_tls cert-manager
check "serving dnsNames" "$(q "$SERVING | .spec.dnsNames[] | select(. == \"$F.cert-manager.svc.cluster.local\")")" \
	"$F.cert-manager.svc.cluster.local"
SELECTOR='{"app.kubernetes.io/instance":"yandex","app.kubernetes.io/name":"cert-manager-webhook-yandex"}'
check "Deployment selector" "$(qj 'select(.kind == "Deployment") | .spec.selector.matchLabels | sort_keys(.)')" "$SELECTOR"
check "Service selector" "$(qj "$SVC | .selector | sort_keys(.)")" "$SELECTOR"
check "documents without helm.sh/chart label" \
	"$(q 'select(.metadata.labels."helm.sh/chart" != "cert-manager-webhook-yandex-0.1.0") | .kind + "/" + .metadata.name')" ""
check "domain-solver subject" "$(subjects ClusterRoleBinding "$F:domain-solver")" "cert-manager/cert-manager"
check "secrets-reader rules" "$(qj "select(.kind == \"ClusterRole\" and .metadata.name == \"$F:secrets-reader\") | .rules")" \
	'[{"apiGroups":[""],"resources":["secrets"],"verbs":["get"]},{"apiGroups":["flowcontrol.apiserver.k8s.io"],"resources":["flowschemas","prioritylevelconfigurations"],"verbs":["list","watch"]}]'

case_name=samename
render cert-manager-webhook-yandex cert-manager
check "Deployment name" "$(q 'select(.kind == "Deployment") | .metadata.name')" "cert-manager-webhook-yandex"
check_tls cert-manager

case_name=fullname
render yandex cert-manager --set fullnameOverride=wh
check "Deployment name" "$(q 'select(.kind == "Deployment") | .metadata.name')" "wh"
check_tls cert-manager

case_name=ns
render yandex webhooks
check "ServiceAccount namespace" "$(q 'select(.kind == "ServiceAccount") | .metadata.namespace')" "webhooks"
check "secrets-reader subject" "$(subjects ClusterRoleBinding "$F:secrets-reader")" "webhooks/$F"
check "auth-delegator subject" "$(subjects ClusterRoleBinding "$F:auth-delegator")" "webhooks/$F"
check "authentication-reader subject" "$(subjects RoleBinding "$F:webhook-authentication-reader")" "webhooks/$F"
check "namespaced resources" \
	"$(q 'select(.kind == "Deployment" or .kind == "Service" or .kind == "Issuer" or .kind == "Certificate") | .metadata.namespace' | sort -u)" "webhooks"
check "APIService service.namespace" "$(q "$API | .spec.service.namespace")" "webhooks"
check "APIService inject-ca-from" "$(q "$API | .metadata.annotations.\"cert-manager.io/inject-ca-from\"")" "webhooks/$F-tls"
check "domain-solver subject" "$(subjects ClusterRoleBinding "$F:domain-solver")" "cert-manager/cert-manager"

case_name=longname
# cert-manager rejects a commonName over 64 bytes.
render a-release-name-long-enough-to-overflow-the-ca-cn cert-manager
check "CA commonName within 64 bytes" "$(q 'select(.kind == "Certificate" and .spec.isCA == true) | .spec.commonName | length <= 64')" "true"

case_name=verbosity0
render yandex cert-manager --set logging.debug=true,logging.verbosity=0
check "last arg" "$(q "$CTR | .args[-1]")" "--v=0"

case_name=emptyformat
render yandex cert-manager --set logging.format=
check "--logging-format" "$(q "$CTR | .args[] | select(test(\"^--logging-format=\"))")" "--logging-format=json-rfc3339"

case_name=debug
render yandex cert-manager --set logging.debug=true
check "last arg" "$(q "$CTR | .args[-1]")" "--v=1"

case_name=pullsecret
render yandex cert-manager --set 'imagePullSecrets[0].name=regcred'
check "imagePullSecrets" "$(qj "$DEP | .imagePullSecrets")" '[{"name":"regcred"}]'

case_name=api
render yandex cert-manager --set customCloudApi.enabled=true
check "API_ENDPOINT" "$(q "$CTR | .env[] | select(.name == \"API_ENDPOINT\") | .value")" "api.cloud.yandex.net:443"

case_name=svcport
render yandex cert-manager --set service.port=8443
check "Service port" "$(q "$SVC | .ports[0].port")" "8443"
check "APIService service.port" "$(q "$API | .spec.service.port")" "8443"

case_name=cport
render yandex cert-manager --set containerPort=7443
check "--secure-port" "$(q "$CTR | .args[] | select(test(\"^--secure-port=\"))")" "--secure-port=7443"
check "containerPort" "$(q "$CTR | .ports[0].containerPort")" "7443"

case_name=cm
render yandex cert-manager --set certManager.namespace=cm,certManager.serviceAccountName=cm-sa
check "domain-solver subject" "$(subjects ClusterRoleBinding "$F:domain-solver")" "cm/cm-sa"

if ((failures > 0)); then
	echo "$failures assertion(s) failed"
	exit 1
fi
echo "all chart assertions passed"
