# Yandex Cloud DNS ACME webhook

A cert-manager DNS01 webhook that solves ACME challenges by creating TXT records in Yandex Cloud DNS. The Helm chart in `deploy/cert-manager-webhook-yandex` deploys only the webhook; cert-manager itself and all Issuers are installed and managed by the operator. This document is for operators installing the webhook and for maintainers publishing releases.

## Prerequisites

* cert-manager installed from the jetstack chart (`https://charts.jetstack.io`) with cainjector enabled (the default); the webhook's APIService gets its CA through `cert-manager.io/inject-ca-from`.
* Helm 3 or later.
* A Yandex Cloud service account with permission to create and delete records in the DNS zone.

## Install

```shell
git clone https://github.com/drengskapr/cert-manager-webhook-yandex.git
helm install -n cert-manager yandex-webhook ./cert-manager-webhook-yandex/deploy/cert-manager-webhook-yandex
```

`certManager.namespace` and `certManager.serviceAccountName` must match the cert-manager installation; the defaults match the jetstack chart's defaults (`cert-manager` / `cert-manager`). The release can live in another namespace: the webhook's own resources follow `--namespace`.

The chart creates no `Issuer`, `ClusterIssuer` or credentials `Secret`. Its internal Issuers and Certificates only sign the webhook's serving certificate.

## Service account key Secret

Create an [authorized key](https://yandex.cloud/en/docs/iam/operations/iam-token/create-for-sa#via-cli) for the service account:

```shell
yc iam key create iamkey \
  --service-account-id=<your service account ID> \
  --format=json \
  --output=iamkey.json
```

Store it in a Secret in the namespace of the Issuer that uses it. For a `ClusterIssuer`, use cert-manager's cluster resource namespace (`cert-manager` by default):

```shell
kubectl create secret generic cert-manager-secret --from-file=iamkey.json -n cert-manager
```

## Issuers

Apply an [Issuer or ClusterIssuer](https://cert-manager.io/docs/configuration/acme/) with a [webhook solver](https://cert-manager.io/docs/configuration/acme/dns01/webhook/) after installing the chart. The solver takes:

| Field | Value |
|---|---|
| `groupName` | The chart's `groupName` value, `acme.cloud.yandex.com` by default |
| `solverName` | `yandex-cloud-dns` |
| `config.folder` | ID of the folder that holds the DNS zone |
| `config.serviceAccountSecretRef` | `name` and `key` of the service account key Secret |

Issuer:

```yaml
apiVersion: cert-manager.io/v1
kind: Issuer
metadata:
  name: issuer
  namespace: default
spec:
  acme:
    email: your@email.com
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: secret-ref
    solvers:
      - dns01:
          webhook:
            config:
              folder: <your folder ID>
              serviceAccountSecretRef:
                name: cert-manager-secret
                key: iamkey.json
            groupName: acme.cloud.yandex.com
            solverName: yandex-cloud-dns
```

ClusterIssuer:

```yaml
apiVersion: cert-manager.io/v1
kind: ClusterIssuer
metadata:
  name: clusterissuer
spec:
  acme:
    email: your@email.com
    server: https://acme-staging-v02.api.letsencrypt.org/directory
    privateKeySecretRef:
      name: secret-ref
    solvers:
      - dns01:
          webhook:
            config:
              folder: <your folder ID>
              serviceAccountSecretRef:
                name: cert-manager-secret
                key: iamkey.json
            groupName: acme.cloud.yandex.com
            solverName: yandex-cloud-dns
```

Certificate with the Issuer:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-com
  namespace: default
spec:
  secretName: example-com-secret
  issuerRef:
    name: issuer
    kind: Issuer
  dnsNames:
    - example.com
```

Certificate with the ClusterIssuer:

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-com
  namespace: default
spec:
  secretName: example-com-secret
  issuerRef:
    name: clusterissuer
    kind: ClusterIssuer
  dnsNames:
    - example.com
```

## Logging

The default format `json-rfc3339` writes one JSON object per line with `ts` as an RFC3339Nano string:

```json
{"ts":"2026-10-30T12:46:13.398936Z","caller":"cert-manager-webhook-yandex/main.go:129","msg":"Present succeeded","v":0,"fqdn":"_acme-challenge.example.com.","zone":"example.com."}
```

`json` is the same format with `ts` in epoch milliseconds; `text` is klog's plain format. `logging.debug: true` adds `--v=<logging.verbosity>`. Logs never contain the challenge key or Secret data.

## Image and releases

The image `ghcr.io/drengskapr/cert-manager-webhook-yandex` is public and built by `.github/workflows/release.yml`. Pushes to branches other than `main` build it without publishing; `v*` tag pushes publish it.

1. Set `version` and `appVersion` in `deploy/cert-manager-webhook-yandex/Chart.yaml` to `X.Y.Z` and commit.
2. Push the tag `vX.Y.Z`; the workflow publishes the image as `X.Y.Z`.
3. After the first publish only, switch the GHCR package to public in its package settings: GHCR creates user-scoped packages as private, and `GITHUB_TOKEN` cannot change visibility.

## Tests

| Command | Covers |
|---|---|
| `go test ./...` | Unit tests |
| `scripts/test_chart.sh` | Rendered chart manifests; needs `helm` and `yq` v4 |
| `TEST_ZONE_NAME=example.com. go test -tags conformance -run TestRunsSuite ./...` | cert-manager DNS01 conformance suite; needs envtest binaries (`TEST_ASSET_ETCD`, `TEST_ASSET_KUBE_APISERVER`, `TEST_ASSET_KUBECTL`) and the files described in `testdata/yandex-cloud-dns/README.md` |

## Values

| Key | Default | Description |
|---|---|---|
| `image.repository` | `ghcr.io/drengskapr/cert-manager-webhook-yandex` | Webhook image repository |
| `image.tag` | `""` | Image tag; empty renders `.Chart.AppVersion` |
| `image.pullPolicy` | `IfNotPresent` | Image pull policy |
| `imagePullSecrets` | `[]` | Pull secrets for the webhook pod, e.g. `[{name: regcred}]` |
| `nameOverride` | `""` | Overrides the chart name in resource names and labels |
| `fullnameOverride` | `""` | Overrides the full resource name |
| `groupName` | `acme.cloud.yandex.com` | API group served by the webhook; Issuers reference it as `webhook.groupName` |
| `customCloudApi.enabled` | `false` | Render the `API_ENDPOINT` env var |
| `customCloudApi.endpoint` | `api.cloud.yandex.net:443` | Yandex Cloud API endpoint |
| `logging.format` | `json-rfc3339` | `json-rfc3339`, `json` or `text` |
| `logging.debug` | `false` | Render `--v=<logging.verbosity>` |
| `logging.verbosity` | `1` | Verbosity level, applied only when `logging.debug` is true |
| `certManager.namespace` | `cert-manager` | Namespace of the cert-manager installation |
| `certManager.serviceAccountName` | `cert-manager` | ServiceAccount of the cert-manager controller |
| `replicaCount` | `1` | Webhook pod replicas |
| `service.type` | `ClusterIP` | Service type |
| `service.port` | `443` | Service port, also used by the APIService |
| `containerPort` | `6443` | Port the webhook container listens on |
| `resources` | `{}` | Webhook container resources |
| `podSecurityContext` | `runAsNonRoot: true`, `runAsUser: 65532`, `runAsGroup: 65532`, `seccompProfile.type: RuntimeDefault` | Pod security context; numeric IDs are required because the image's user is non-numeric (`nonroot`) |
| `securityContext` | `allowPrivilegeEscalation: false`, `readOnlyRootFilesystem: true`, `capabilities.drop: [ALL]` | Container security context |
| `nodeSelector` | `{}` | Node selector for the webhook pod |
| `tolerations` | `[]` | Tolerations for the webhook pod |
| `affinity` | `{}` | Affinity for the webhook pod |
