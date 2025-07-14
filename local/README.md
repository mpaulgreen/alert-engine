# Alert Engine Local Development

This directory contains all files needed for local development of the Alert Engine while connecting to OpenShift infrastructure.

## Files

- **`LOCAL_SETUP_GUIDE.md`** - Comprehensive setup guide with detailed instructions
- **`QUICK_START.md`** - Quick reference for 3-step setup
- **`local-setup.sh`** - Automated setup script that handles prerequisites, builds the binary, and creates configuration
- **`test-local-setup.sh`** - Validation script that runs 10 tests to verify your setup is working

## Quick Start

```bash
# Navigate to this directory
cd alert-engine/local

# Run automated setup
./local-setup.sh

# Test the setup
./test-local-setup.sh
```

## Directory Structure

```
alert-engine/
├── local/                          # 👈 You are here
│   ├── LOCAL_SETUP_GUIDE.md        # Detailed setup guide
│   ├── QUICK_START.md              # Quick reference
│   ├── local-setup.sh              # Automated setup
│   ├── test-local-setup.sh         # Validation tests
│   └── README.md                   # This file
├── cmd/                            # Source code
├── configs/                        # Configuration files
└── ... (other project files)
```

## Prerequisites

- Go 1.23+
- OpenShift CLI (oc) configured
- Access to OpenShift cluster with deployed infrastructure

## Next Steps

1. Follow the **[QUICK_START.md](QUICK_START.md)** for a 3-step setup
2. Read **[LOCAL_SETUP_GUIDE.md](LOCAL_SETUP_GUIDE.md)** for detailed instructions
3. Use **`./local-setup.sh`** to automate the setup process
4. Validate with **`./test-local-setup.sh`** to ensure everything works

For troubleshooting, see the detailed guide. 