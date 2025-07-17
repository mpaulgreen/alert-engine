# Alert Engine Container Build System

## 📋 Overview

This document summarizes the complete container build system created for the Alert Engine, following Red Hat standards and OpenShift best practices.

## 📁 Created Files

### Core Build Files

| File | Purpose | Red Hat Compliance |
|------|---------|-------------------|
| **`Dockerfile`** | Multi-stage container build | ✅ UBI8 base images, security hardened |
| **`build.sh`** | Comprehensive build script | ✅ OpenShift compatibility, automated workflows |
| **`../Makefile`** | Development workflow automation | ✅ Standard build patterns |
| **`../.dockerignore`** | Optimized build context | ✅ Efficient builds |

### Integration with Deployment

| File | Auto-Updated | Purpose |
|------|--------------|---------|
| `deployment.yaml` | ✅ Yes | Container image reference |
| `kustomization.yaml` | ✅ Yes | Kustomize image config |
| `README.md` | 📝 Manual | Build documentation |

## 🏗️ Build Architecture

### Multi-Stage Dockerfile

```dockerfile
# Stage 1: Builder (UBI8 Go Toolset)
FROM registry.access.redhat.com/ubi8/go-toolset:latest AS builder
# - Downloads dependencies
# - Compiles static Go binary
# - Embeds version and build time

# Stage 2: Runtime (UBI8 Micro)  
FROM registry.access.redhat.com/ubi8/ubi-micro:latest
# - Minimal attack surface (~50MB)
# - Non-root user (UID 1001)
# - Security hardened
```

### Key Features

- ✅ **Red Hat UBI8** compliance
- ✅ **Multi-stage** for minimal size
- ✅ **Static binary** (CGO_ENABLED=0)
- ✅ **Version embedding** at build time
- ✅ **Health checks** built-in
- ✅ **OpenShift SCC** compatible

## 🚀 Usage Examples

### Quick Start

```bash
# Build locally
./deployments/alert-engine/build.sh

# Build and push with version
./deployments/alert-engine/build.sh --version v1.0.0 --push

# Full release build with tests
make release-build
```

### Development Workflow

```bash
# Set up development environment
make dev-setup

# Run quick checks
make dev-test

# Build and test container
make docker-test

# Deploy locally
make deploy-local
```

### Production Build

```bash
# Full production build with all checks
./deployments/alert-engine/build.sh \
  --version v1.2.3 \
  --registry quay.io/your-org \
  --test \
  --push

# Results in:
# - quay.io/your-org/alert-engine:v1.2.3
# - Updated deployment manifests
# - Ready for production deployment
```

## 🔧 Build Script Capabilities

### Command Line Options

| Option | Description | Example |
|--------|-------------|---------|
| `--push` | Push to registry | `--push` |
| `--tag` | Specify image tag | `--tag v1.0.0` |
| `--registry` | Set registry | `--registry quay.io/myorg` |
| `--version` | Embed version | `--version 1.2.3` |
| `--test` | Run tests first | `--test` |
| `--no-cache` | Force rebuild | `--no-cache` |
| `--dry-run` | Show commands | `--dry-run` |

### Automated Features

- 🔍 **Prerequisites check** - Validates tools and files
- 🧪 **Optional testing** - Runs Go tests before build
- 🏗️ **Multi-platform** - Builds for x86_64 (OpenShift standard)
- 📝 **Manifest updates** - Updates deployment files automatically
- 🎨 **Colored output** - Clear build progress indication

## 📊 Container Specifications

### Image Characteristics

| Aspect | Value |
|--------|-------|
| **Base Runtime** | UBI8 Micro |
| **Size** | ~50MB |
| **Architecture** | linux/amd64 |
| **User** | Non-root (UID 1001) |
| **Security** | Hardened, minimal surface |

### Security Features

- ✅ **Non-root execution** (UID 1001)
- ✅ **Read-only filesystem** ready
- ✅ **Minimal base image** (UBI8 Micro)
- ✅ **No package manager** in runtime
- ✅ **Static binary** - no dependencies
- ✅ **Capabilities dropped** - all unnecessary capabilities removed

### Health & Monitoring

- ✅ **Health check** endpoint (`/health`)
- ✅ **Readiness check** endpoint (`/ready`)  
- ✅ **Metrics endpoint** (`/metrics`)
- ✅ **Prometheus** annotations
- ✅ **Structured logging** support

## 🔄 CI/CD Integration

### GitHub Actions Ready

```yaml
- name: Build and Push
  run: |
    ./deployments/alert-engine/build.sh \
      --version ${{ github.ref_name }} \
      --test \
      --push
```

### OpenShift Build Integration

```yaml
# BuildConfig example
source:
  git:
    uri: https://github.com/your-org/alert-engine
strategy:
  dockerStrategy:
    dockerfilePath: deployments/alert-engine/Dockerfile
```

## 📈 Performance Optimizations

### Build Context

- **`.dockerignore`** - Excludes unnecessary files
- **Layer caching** - Go modules cached separately
- **Multi-stage** - Only runtime artifacts in final image
- **Static binary** - No dynamic linking overhead

### Runtime Efficiency

- **UBI8 Micro** - Minimal base image
- **Single binary** - No installation required
- **Health checks** - Fast startup detection
- **Resource requests** - Conservative defaults

## 🏆 Comparison with Mock Log Generator

| Aspect | Mock Log Generator | Alert Engine |
|--------|-------------------|--------------|
| **Language** | Python 3.9 | Go (latest) |
| **Base Image** | UBI8 Python | UBI8 Go Toolset → UBI8 Micro |
| **Final Size** | ~200MB | ~50MB |
| **Dependencies** | requirements.txt | go.mod |
| **Build Type** | Single-stage | Multi-stage |
| **Security** | Standard | Enhanced (static binary) |

## 📚 References

- [Red Hat Universal Base Images](https://catalog.redhat.com/software/containers/ubi8)
- [OpenShift Container Platform Documentation](https://docs.openshift.com/)
- [Go Multi-stage Builds](https://docs.docker.com/build/building/multi-stage/)
- [Container Security Best Practices](https://cloud.google.com/architecture/best-practices-for-operating-containers)

---

**Ready for Production Deployment!** 🚀

The Alert Engine now has a complete, production-ready container build system that follows Red Hat standards and OpenShift best practices. 