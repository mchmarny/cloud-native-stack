# Software Supply Chain Security Demo

Demonstration of supply chain security artifacts provided by Cloud Native Stack.

![software supply chain security](images/s3c.png)

## Overview

Cloud Native Stack (CNS) provides supply chain security artifacts:

- **SBOM Attestation**: Complete inventory of packages, libraries, and components in SPDX format
- **SLSA Build Provenance**: Verifiable build information (how and where images were created)
- **Keyless Signing**: Artifacts signed using Sigstore (Fulcio + Rekor)

## Image Attestations

**Build Provenance (SLSA L3)**
- Complete record of the build environment, tools, and process
- Source repository URL and exact commit SHA
- GitHub Actions workflow that produced the artifact
- Build parameters and environment variables
- Cryptographically signed using Sigstore keyless signing

Get latest release tag:

```shell
export TAG=$(curl -s https://api.github.com/repos/NVIDIA/cloud-native-stack/releases/latest | jq -r '.tag_name')
echo "Using tag: $TAG"
```
Resolve tag to immutable digest:

```shell
export IMAGE="ghcr.io/mchmarny/cns"
export DIGEST=$(crane digest "${IMAGE}:${TAG}")
echo "Resolved digest: $DIGEST"
export IMAGE_DIGEST="${IMAGE}@${DIGEST}"
```

> Tags are mutable and can be changed to point to different images. Digests are immutable SHA256 hashes that uniquely identify an image, providing stronger security guarantees.

**Method 1: GitHub CLI (Recommended)**

Verify using digest:

```shell
gh attestation verify oci://${IMAGE_DIGEST} --owner nvidia
```

Verify the cnsd image:

```shell
export IMAGE_API="ghcr.io/mchmarny/cnsd"
export DIGEST_API=$(crane digest "${IMAGE_API}:${TAG}")
gh attestation verify oci://${IMAGE_API}@${DIGEST_API} --owner nvidia
```

**Method 2: Cosign (SBOM Attestations)**

Verify SBOM attestation using digest:

```shell
cosign verify-attestation \
  --type spdxjson \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/NVIDIA/cloud-native-stack/.github/workflows/.*' \
  ${IMAGE_DIGEST}
```


## SBOM

**SBOM Attestations (SPDX v2.3 JSON for Binary & Images)**
- Complete inventory of packages, libraries, and dependencies
- Attached to container images as attestations
- Signed with Cosign using keyless signing (Fulcio + Rekor)
- Enables vulnerability scanning and license compliance

**Access Binary SBOM:**

Get latest release tag:

```shell
export TAG=$(curl -s https://api.github.com/repos/NVIDIA/cloud-native-stack/releases/latest | jq -r '.tag_name')
export VERSION=${TAG#v}  # Remove 'v' prefix for filenames
```

Detect OS and architecture:
```shell
export OS=$(uname -s | tr '[:upper:]' '[:lower:]')
export ARCH=$(uname -m | sed 's/x86_64/amd64/; s/aarch64/arm64/')
```

Download binary from GitHub releases:
```shell
curl -LO https://github.com/NVIDIA/cloud-native-stack/releases/download/${TAG}/cns_${TAG}_${OS}_${ARCH}
chmod +x cns_${TAG}_${OS}_${ARCH}
```

Download SBOM (separate file):
```shell
curl -LO https://github.com/NVIDIA/cloud-native-stack/releases/download/${TAG}/cns_${VERSION}_${OS}_${ARCH}.sbom.json
```

View SBOM
```shell
cat cns_${VERSION}_${OS}_${ARCH}.sbom.json
```

### Container Image SBOMs (API Server & Agent)

Get latest release tag and resolve digest:

```shell
export TAG=$(curl -s https://api.github.com/repos/NVIDIA/cloud-native-stack/releases/latest | jq -r '.tag_name')
export IMAGE="ghcr.io/mchmarny/cnsd"
export DIGEST=$(crane digest "${IMAGE}:${TAG}")
export IMAGE_DIGEST="${IMAGE}@${DIGEST}"
```

*Method 1*: Using Cosign (extracts attestation) - uses digest to avoid warnings:

```shell
cosign verify-attestation \
  --type spdxjson \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/NVIDIA/cloud-native-stack/.github/workflows/.*' \
  ${IMAGE_DIGEST} | \
  jq -r '.payload' | base64 -d | jq '.predicate' > sbom.json
```

*Method 2*: Using GitHub CLI (shows all attestations)
```shell
gh attestation verify oci://${IMAGE_DIGEST} --owner nvidia --format json
```

**SBOM Use Cases:**

1. **Vulnerability Scanning** – Feed SBOM to Grype, Trivy, or Snyk
   ```shell
   grype sbom:./sbom.json
   ```

2. **License Compliance** – Analyze licensing obligations
   ```shell
   jq -r '.packages[] | select(.licenseDeclared != "NOASSERTION") | "\(.name) \(.versionInfo) \(.licenseDeclared)"' sbom.json
   ```

3. **Dependency Tracking** – Monitor for supply chain risks
   ```shell
   jq '.packages[] | select(.name | contains("vulnerable-lib"))' sbom.json
   ```

4. **Audit Trail** – Maintain records for compliance
   ```shell
   # SBOM timestamp proves when components were included
   jq '.creationInfo.created' sbom.json
   ```

### In-Cluster Verification

Enforce provenance verification at deployment time using Kubernetes admission controllers.

**Option 1: Sigstore Policy Controller**

Install Policy Controller:

```shell
kubectl apply -f https://github.com/sigstore/policy-controller/releases/download/v0.10.0/release.yaml
```
Create ClusterImagePolicy to enforce provenance:

```shell
cat <<EOF | kubectl apply -f -
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: cns-images-require-attestation
spec:
  images:
  - glob: "ghcr.io/mchmarny/cns*"
  authorities:
  - keyless:
      url: https://fulcio.sigstore.dev
      identities:
      - issuerRegExp: ".*\.github\.com.*"
        subjectRegExp: "https://github.com/mchmarny/cloud-native-stack/.*"
    attestations:
    - name: build-provenance
      predicateType: https://slsa.dev/provenance/v1
      policy:
        type: cue
        data: |
          predicate: buildDefinition: buildType: "https://actions.github.io/buildtypes/workflow/v1"
EOF
```

**Option 2: Kyverno Policy**

```yaml
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: verify-cns-attestations
spec:
  validationFailureAction: Enforce
  rules:
  - name: verify-attestation
    match:
      any:
      - resources:
          kinds:
          - Pod
    verifyImages:
    - imageReferences:
      - "ghcr.io/mchmarny/cns*"
      attestations:
      - predicateType: https://slsa.dev/provenance/v1
        attestors:
        - entries:
          - keyless:
              issuer: https://token.actions.githubusercontent.com
              subject: https://github.com/mchmarny/cloud-native-stack/.github/workflows/*
```

**Test Policy Enforcement:**

Get latest release tag:

```shell
export TAG=$(curl -s https://api.github.com/repos/mchmarny/cloud-native-stack/releases/latest | jq -r '.tag_name')
```

This should succeed (image with valid attestation):

```shell
kubectl run test-valid --image=ghcr.io/mchmarny/cns:${TAG}
```
This should fail (unsigned image):

```shell
kubectl run test-invalid --image=nginx:latest
```

> Error: image verification failed: no matching attestations found

#### Build Process Transparency

All CNS releases are built using GitHub Actions with full transparency:

1. **Source Code** – Public GitHub repository
2. **Build Workflow** – `.github/workflows/on-tag.yaml` (version controlled)
3. **Build Logs** – Public GitHub Actions run logs
4. **Attestations** – Signed and stored in public transparency log (Rekor)
5. **Artifacts** – Published to GitHub Releases and GHCR

**View Build History:**

List all releases with attestations:

```shell
gh api repos/NVIDIA/cloud-native-stack/releases | \
  jq -r '.[] | "\(.tag_name): \(.html_url)"'
```

View specific build logs:

```shell
gh run list --repo NVIDIA/cloud-native-stack --workflow=on-tag.yaml
gh run view 20642050863 --repo NVIDIA/cloud-native-stack --log
```

**Verify in Transparency Log (Rekor):**

Search Rekor for attestations:

```shell
rekor-cli search --artifact ghcr.io/mchmarny/cns:v0.8.12
```

Get entry details:

```shell
rekor-cli get --uuid <entry-uuid>
```
