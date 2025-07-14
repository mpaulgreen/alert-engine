# Alert Engine Local Development

This directory contains all files needed for local development of the Alert Engine while connecting to OpenShift infrastructure.

## Files

- **`LOCAL_SETUP_GUIDE.md`** - Comprehensive setup guide with detailed instructions
- **`local-setup.sh`** - Automated setup script that handles prerequisites, builds the binary, and creates configuration

## Quick Start

```bash
# Navigate to this directory
cd alert-engine/local

# Run automated setup
./local-setup.sh
```

## Directory Structure

```
alert-engine/
├── local/                          # 👈 You are here
│   ├── LOCAL_SETUP_GUIDE.md        # Detailed setup guide
│   ├── local-setup.sh              # Automated setup
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

1. Read **[LOCAL_SETUP_GUIDE.md](LOCAL_SETUP_GUIDE.md)** for detailed instructions
2. Use **`./local-setup.sh`** to automate the setup process

For troubleshooting, see the detailed guide. 