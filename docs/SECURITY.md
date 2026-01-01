# Security

NVIDIA is dedicated to the security and trust of our software products and services, including all source code repositories.

**Please do not report security vulnerabilities through GitHub.**

## Reporting Security Vulnerabilities

To report a potential security vulnerability in any NVIDIA product:

- **Web**: [Security Vulnerability Submission Form](https://www.nvidia.com/object/submit-security-vulnerability.html)
- **Email**: psirt@nvidia.com
  - Use [NVIDIA PGP Key](https://www.nvidia.com/en-us/security/pgp-key) for secure communication

**Include in your report**:
- Product/Driver name and version
- Type of vulnerability (code execution, denial of service, buffer overflow, etc.)
- Steps to reproduce
- Proof-of-concept or exploit code
- Potential impact and exploitation method

NVIDIA offers acknowledgement for externally reported security issues under our coordinated vulnerability disclosure policy. Visit [PSIRT Policies](https://www.nvidia.com/en-us/security/psirt-policies/) for details.

## Product Security Resources

For all security-related concerns: https://www.nvidia.com/en-us/security

## Supply Chain Security

Cloud Native Stack (CNS) provides supply chain security artifacts for all container images:

- **SBOM Attestation**: Complete inventory of packages, libraries, and components
- **SLSA Build Provenance**: Verifiable build information (how and where images were created)

### Container Image Attestations

All container images published from tagged releases include **multiple layers of attestations**, providing comprehensive supply chain security:

1. **Build Provenance** – SLSA attestations signed using GitHub's OIDC identity
2. **SBOM Attestations** – CycloneDX format signed with Cosign
3. **Binary SBOMs** – Embedded in CLI binaries via GoReleaser

#### Attestation Types

**Build Provenance (SLSA)**
- Complete record of the build environment, tools, and process
- Source repository URL and exact commit SHA
- GitHub Actions workflow that produced the artifact
- Build parameters and environment variables
- Cryptographically signed using Sigstore keyless signing
- SLSA Build Level 3 compliant

**SBOM Attestations**
- Complete inventory of packages, libraries, and dependencies
- CycloneDX JSON format (industry standard)
- Attached to container images as attestations
- Signed with Cosign using keyless signing (Fulcio + Rekor)
- Enables vulnerability scanning and license compliance

#### Verify Image Attestations

**Method 1: GitHub CLI (Recommended)**

```shell
# Verify the eidos CLI image
gh attestation verify oci://ghcr.io/mchmarny/eidos:v0.8.10 --owner mchmarny

# Verify the eidos-api-server image  
gh attestation verify oci://ghcr.io/mchmarny/eidos-api-server:v0.8.10 --owner mchmarny

# Verify with specific digest for immutability
gh attestation verify oci://ghcr.io/mchmarny/eidos@sha256:abc123... --owner mchmarny
```

**Method 2: Cosign (SBOM Attestations)**

```shell
# Verify SBOM attestation signature
cosign verify-attestation \
  --type cyclonedx \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/mchmarny/cloud-native-stack/.github/workflows/.*' \
  ghcr.io/mchmarny/eidos:v0.8.10

# Extract and view SBOM
cosign verify-attestation \
  --type cyclonedx \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/mchmarny/cloud-native-stack/.github/workflows/.*' \
  ghcr.io/mchmarny/eidos:v0.8.10 | jq -r '.payload' | base64 -d | jq '.predicate'
```

**Method 3: Policy Enforcement (Kubernetes)**

See [In-Cluster Verification](#in-cluster-verification) section below for automated admission policies.

Replace `TAG` with the specific version you want to verify (e.g., `v0.8.10`).

#### What's Included in Attestations

**Build Provenance Contains:**
- Build trigger (tag push event)
- Builder identity (GitHub Actions runner)
- Source repository and commit SHA
- Build workflow path and run ID
- Build parameters and environment
- Dependencies used during build
- Timestamp and build duration

**SBOM Contains:**
- All Go module dependencies with versions
- Transitive dependencies (full dependency tree)
- Package licenses (SPDX identifiers)
- Package URLs (purl) for each component
- Container base image layers
- System packages from base image

For more information:
- [GitHub Artifact Attestations](https://docs.github.com/en/actions/security-for-github-actions/using-artifact-attestations)
- [SLSA Framework](https://slsa.dev/)
- [CycloneDX SBOM Standard](https://cyclonedx.org/)
- [Sigstore Cosign](https://docs.sigstore.dev/cosign/overview/)

### Setup

Export variables for the image you want to verify, for example:

```shell
export IMAGE="ghcr.io/mchmarny/eidos"
export DIGEST="sha256:f33ee5ed8372e1d989ac858e7319b5ad5c64318683a79f1c8c20b1bbedf3e724"
export IMAGE_DIGEST="$IMAGE@$DIGEST"
export IMAGE_SBOM="$IMAGE:sha256-$(echo "$DIGEST" | cut -d: -f2).sbom"
```

**Authentication** (if needed):
```shell
docker login ghcr.io
```

### Software Bill of Materials (SBOM)

Cloud Native Stack provides **two types of SBOMs** for comprehensive supply chain visibility:

1. **Binary SBOMs** – Embedded in CLI binaries (SPDX v2.3 format)
2. **Container Image SBOMs** – Attached as attestations (CycloneDX JSON format)

#### Binary SBOMs (CLI)

Generated by GoReleaser during release builds, embedded directly in binaries.

**Access Binary SBOM:**

```shell
# Download binary from GitHub releases
curl -LO https://github.com/mchmarny/cloud-native-stack/releases/download/v0.8.10/eidos_v0.8.10_darwin_arm64
chmod +x eidos_v0.8.10_darwin_arm64

# Download SBOM (separate file)
curl -LO https://github.com/mchmarny/cloud-native-stack/releases/download/v0.8.10/eidos_0.8.10_darwin_arm64.sbom.json

# View SBOM
cat eidos_0.8.10_darwin_arm64.sbom.json
```

**Binary SBOM Format** (SPDX v2.3):

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "eidos",
  "documentNamespace": "https://anchore.com/syft/file/eidos-610e106b-2614-434c-bfe6-941863de47ff",
  "creationInfo": {
    "licenseListVersion": "3.27",
    "creators": [
      "Organization: Anchore, Inc",
      "Tool: syft-1.38.2"
    ],
    "created": "2026-01-01T16:52:12Z"
  },
  "packages": [
    {
      "name": "github.com/NVIDIA/cloud-native-stack",
      "SPDXID": "SPDXRef-Package-go-module-github.com-NVIDIA-cloud-native-stack-f06a66ba03567417",
      "versionInfo": "v0.8.10",
      "supplier": "NOASSERTION",
      "downloadLocation": "NOASSERTION",
      "filesAnalyzed": false,
      "sourceInfo": "acquired package info from go module information: /eidos",
      "licenseConcluded": "NOASSERTION",
      "licenseDeclared": "NOASSERTION",
      "copyrightText": "NOASSERTION",
      "externalRefs": [
        {
          "referenceCategory": "SECURITY",
          "referenceType": "cpe23Type",
          "referenceLocator": "cpe:2.3:a:NVIDIA:cloud-native-stack:v0.8.10:*:*:*:*:*:*:*"
        },
        {
          "referenceCategory": "SECURITY",
          "referenceType": "cpe23Type",
          "referenceLocator": "cpe:2.3:a:NVIDIA:cloud_native_stack:v0.8.10:*:*:*:*:*:*:*"
        },
        {
          "referenceCategory": "PACKAGE-MANAGER",
          "referenceType": "purl",
          "referenceLocator": "pkg:golang/github.com/NVIDIA/cloud-native-stack@v0.8.10"
        }
      ]
    },
```

#### Container Image SBOMs (API Server & Agent)

Generated by Syft/Anchore, attached as Cosign attestations in CycloneDX format.

**Extract SBOM from Container Image:**

```shell
# Method 1: Using Cosign (extracts attestation)
cosign verify-attestation \
  --type cyclonedx \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  --certificate-identity-regexp 'https://github.com/mchmarny/cloud-native-stack/.github/workflows/.*' \
  ghcr.io/mchmarny/eidos-api-server:v0.8.10 | \
  jq -r '.payload' | base64 -d | jq '.predicate' > sbom.json

# Method 2: Using GitHub CLI (shows all attestations)
gh attestation verify oci://ghcr.io/mchmarny/eidos-api-server:v0.8.10 --owner mchmarny --format json
```

**Container SBOM Format** (CycloneDX JSON):

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "eidos",
  "documentNamespace": "https://anchore.com/syft/file/eidos-610e106b-2614-434c-bfe6-941863de47ff",
  "creationInfo": {
    "licenseListVersion": "3.27",
    "creators": [
      "Organization: Anchore, Inc",
      "Tool: syft-1.38.2"
    ],
    "created": "2026-01-01T16:52:12Z"
  },
  "packages": [
    {
      "name": "github.com/NVIDIA/cloud-native-stack",
      "SPDXID": "SPDXRef-Package-go-module-github.com-NVIDIA-cloud-native-stack-f06a66ba03567417",
      "versionInfo": "v0.8.10",
      ...
```

**SBOM Use Cases:**

1. **Vulnerability Scanning** – Feed SBOM to Grype, Trivy, or Snyk
   ```shell
   grype sbom:./sbom.json
   ```

2. **License Compliance** – Analyze licensing obligations
   ```shell
   jq '[.components[] | {name, version, license: .licenses[0].license.id}]' sbom.json
   ```

3. **Dependency Tracking** – Monitor for supply chain risks
   ```shell
   jq '.components[] | select(.name | contains("vulnerable-lib"))' sbom.json
   ```

4. **Audit Trail** – Maintain records for compliance
   ```shell
   # SBOM timestamp proves when components were included
   jq '.metadata.timestamp' sbom.json
   ```
### SLSA Build Provenance

SLSA (Supply chain Levels for Software Artifacts) provides verifiable information about how images were built. Cloud Native Stack achieves **SLSA Build Level 3** through GitHub Actions OIDC integration.

#### What is SLSA?

SLSA is a security framework that protects against supply chain attacks by ensuring:
- **Source integrity** – Code comes from expected repository
- **Build integrity** – Build process is secure and reproducible
- **Provenance** – Complete record of how artifacts were created
- **Auditability** – Cryptographically signed evidence

#### SLSA Level 3 Requirements (Achieved)

✅ **Build as Code** – GitHub Actions workflows define reproducible builds  
✅ **Provenance Available** – Attestations generated for all releases  
✅ **Provenance Authenticated** – Signed using Sigstore keyless signing  
✅ **Service Generated** – GitHub Actions generates provenance (not self-asserted)  
✅ **Non-falsifiable** – Strong authentication of identity (OIDC)  
✅ **Dependencies Complete** – Full dependency graph in SBOM  

#### Verify SLSA Provenance

**Method 1: GitHub CLI**

```shell
# Verify provenance exists and is valid
gh attestation verify oci://ghcr.io/mchmarny/eidos:v0.8.10 --owner mchmarny

# Output shows:
# ✓ Verification succeeded!
# 
# Attestations:
#   • Build provenance (SLSA v1.0)
#   • SBOM (CycloneDX)
```

**Method 2: Extract and Inspect Provenance**

```shell
# Get full provenance data
gh attestation verify oci://ghcr.io/mchmarny/eidos:v0.8.10 \
  --owner mchmarny \
  --format json | jq '.[] | select(.verificationResult.statement.predicateType | contains("slsa"))'

# Key fields in provenance:
# - buildDefinition.buildType: GitHub Actions workflow type
# - runDetails.builder.id: Workflow file and commit
# - buildDefinition.externalParameters.workflow: Workflow path and ref
# - buildDefinition.resolvedDependencies: Source code commit SHA
# - runDetails.metadata.invocationId: GitHub run ID
```

**Example Provenance Data:**

```json
...
  "verificationResult": {
    "mediaType": "application/vnd.dev.sigstore.verificationresult+json;version=0.1",
    "signature": {
      "certificate": {
        "certificateIssuer": "CN=sigstore-intermediate,O=sigstore.dev",
        "subjectAlternativeName": "https://github.com/mchmarny/cloud-native-stack/.github/workflows/on-tag.yaml@refs/tags/v0.8.10",
        "issuer": "https://token.actions.githubusercontent.com",
        "githubWorkflowTrigger": "push",
        "githubWorkflowSHA": "ba6cbbe8b1a8fc8b72bb18454c10a3ba31d94a2e",
        "githubWorkflowName": "on_tag",
        "githubWorkflowRepository": "mchmarny/cloud-native-stack",
        "githubWorkflowRef": "refs/tags/v0.8.10",
        "buildSignerURI": "https://github.com/mchmarny/cloud-native-stack/.github/workflows/on-tag.yaml@refs/tags/v0.8.10",
        "buildSignerDigest": "ba6cbbe8b1a8fc8b72bb18454c10a3ba31d94a2e",
        "runnerEnvironment": "github-hosted",
        "sourceRepositoryURI": "https://github.com/mchmarny/cloud-native-stack",
        "sourceRepositoryDigest": "ba6cbbe8b1a8fc8b72bb18454c10a3ba31d94a2e",
        "sourceRepositoryRef": "refs/tags/v0.8.10",
        "sourceRepositoryIdentifier": "1095163471",
        "sourceRepositoryOwnerURI": "https://github.com/mchmarny",
        "sourceRepositoryOwnerIdentifier": "175854",
        "buildConfigURI": "https://github.com/mchmarny/cloud-native-stack/.github/workflows/on-tag.yaml@refs/tags/v0.8.10",
        "buildConfigDigest": "ba6cbbe8b1a8fc8b72bb18454c10a3ba31d94a2e",
        "buildTrigger": "push",
        "runInvocationURI": "https://github.com/mchmarny/cloud-native-stack/actions/runs/20642050863/attempts/1",
        "sourceRepositoryVisibilityAtSigning": "public"
      }
    },
...
```

#### In-Cluster Verification

Enforce provenance verification at deployment time using Kubernetes admission controllers.

**Option 1: Sigstore Policy Controller**

```shell
# Install Policy Controller
kubectl apply -f https://github.com/sigstore/policy-controller/releases/download/v0.10.0/release.yaml

# Create ClusterImagePolicy to enforce provenance
cat <<EOF | kubectl apply -f -
apiVersion: policy.sigstore.dev/v1beta1
kind: ClusterImagePolicy
metadata:
  name: cns-images-require-attestation
spec:
  images:
  - glob: "ghcr.io/mchmarny/eidos*"
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
      - "ghcr.io/mchmarny/eidos*"
      attestations:
      - predicateType: https://slsa.dev/provenance/v1
        attestors:
        - entries:
          - keyless:
              issuer: https://token.actions.githubusercontent.com
              subject: https://github.com/mchmarny/cloud-native-stack/.github/workflows/*
```

**Test Policy Enforcement:**

```shell
# This should succeed (image with valid attestation)
kubectl run test-valid --image=ghcr.io/mchmarny/eidos:v0.8.10

# This should fail (unsigned image)
kubectl run test-invalid --image=nginx:latest
# Error: image verification failed: no matching attestations found
```

#### Build Process Transparency

All CNS releases are built using GitHub Actions with full transparency:

1. **Source Code** – Public GitHub repository
2. **Build Workflow** – `.github/workflows/on-tag.yaml` (version controlled)
3. **Build Logs** – Public GitHub Actions run logs
4. **Attestations** – Signed and stored in public transparency log (Rekor)
5. **Artifacts** – Published to GitHub Releases and GHCR

**View Build History:**

```shell
# List all releases with attestations
gh api repos/mchmarny/cloud-native-stack/releases | \
  jq -r '.[] | "\(.tag_name): \(.html_url)"'

# View specific build logs
gh run list --repo mchmarny/cloud-native-stack --workflow=on-tag.yaml
gh run view 20642050863 --repo mchmarny/cloud-native-stack --log
```

**Verify in Transparency Log (Rekor):**

```shell
# Search Rekor for attestations
rekor-cli search --artifact ghcr.io/mchmarny/eidos:v0.8.10

# Get entry details
rekor-cli get --uuid <entry-uuid>
```

For more information:
- [SLSA Framework Documentation](https://slsa.dev/)
- [GitHub Actions SLSA Generation](https://github.com/slsa-framework/slsa-github-generator)
- [Sigstore Policy Controller](https://docs.sigstore.dev/policy-controller/overview/)
- [Kyverno Image Verification](https://kyverno.io/docs/writing-policies/verify-images/)
