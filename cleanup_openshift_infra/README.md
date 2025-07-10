# OpenShift Infrastructure Cleanup Tools

This directory contains comprehensive tools for cleaning up the Alert Engine infrastructure from your OpenShift cluster.

## 🛠️ Available Tools

| File | Purpose | Usage |
|------|---------|-------|
| `README_CLEANUP.md` | **Main documentation** | Complete cleanup guide and instructions |
| `verify_resources_before_cleanup.sh` | **Verification script** | Check what resources exist before/after cleanup |
| `cleanup_openshift_infrastructure.sh` | **Automated cleanup** | Main cleanup script - removes all components |
| `manual_cleanup_reference.md` | **Manual commands** | Detailed manual commands for troubleshooting |

## 🚀 Quick Start

1. **Check what exists:**
   ```bash
   ./verify_resources_before_cleanup.sh
   ```

2. **Run automated cleanup:**
   ```bash
   ./cleanup_openshift_infrastructure.sh
   ```

3. **Verify cleanup completed:**
   ```bash
   ./verify_resources_before_cleanup.sh
   ```

## 📋 What Gets Removed

- ✅ Alert Engine applications and configurations
- ✅ Kafka cluster (AMQ Streams) with all data
- ✅ Redis Enterprise cluster with all data  
- ✅ OpenShift Logging and ClusterLogForwarder
- ✅ All operators and CSVs
- ✅ Service accounts, RBAC, and network policies
- ✅ All persistent volumes and data
- ✅ All namespaces: `alert-engine`, `amq-streams-kafka`, `redis-enterprise`, `openshift-logging`

## ⚠️ Important Warning

**This cleanup will permanently delete all data!** Ensure you have backups before proceeding.

## 📚 Documentation

For detailed instructions, troubleshooting, and manual commands, see:
- **[README_CLEANUP.md](README_CLEANUP.md)** - Complete documentation
- **[manual_cleanup_reference.md](manual_cleanup_reference.md)** - Manual commands reference

## 🎯 Usage Location

Run these scripts from any directory. They use `oc` commands to interact with your OpenShift cluster.

---

**Created as part of the Alert Engine OpenShift Infrastructure Setup and Management.** 