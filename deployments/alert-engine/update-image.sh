#!/bin/bash

# Quick script to update image reference in kustomization.yaml
# Usage: ./update-image.sh [IMAGE_NAME] [TAG] [REGISTRY]

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
KUSTOMIZATION_FILE="$SCRIPT_DIR/kustomization.yaml"

# Default values
DEFAULT_REGISTRY="quay.io/mpaulgreen"
DEFAULT_IMAGE_NAME="alert-engine"
DEFAULT_TAG="latest"

# Parse arguments
IMAGE_NAME="${1:-$DEFAULT_IMAGE_NAME}"
TAG="${2:-$DEFAULT_TAG}"
REGISTRY="${3:-$DEFAULT_REGISTRY}"

FULL_IMAGE="$REGISTRY/$IMAGE_NAME"

echo "🔧 Updating image reference in kustomization.yaml"
echo "📦 Image: $FULL_IMAGE:$TAG"

if [[ ! -f "$KUSTOMIZATION_FILE" ]]; then
    echo "❌ kustomization.yaml not found at: $KUSTOMIZATION_FILE"
    exit 1
fi

# Update using yq if available, otherwise use sed
if command -v yq >/dev/null 2>&1; then
    echo "🔨 Using yq to update kustomization.yaml..."
    yq eval ".images[0].name = \"$FULL_IMAGE\"" -i "$KUSTOMIZATION_FILE"
    yq eval ".images[0].newTag = \"$TAG\"" -i "$KUSTOMIZATION_FILE"
    echo "✅ Updated with yq"
else
    echo "🔨 Using sed to update kustomization.yaml..."
    # Create backup
    cp "$KUSTOMIZATION_FILE" "$KUSTOMIZATION_FILE.bak"
    
    # Update name and tag
    sed -i'' "s|name: .*/.*|name: $FULL_IMAGE|g" "$KUSTOMIZATION_FILE"
    sed -i'' "s|newTag: .*|newTag: $TAG|g" "$KUSTOMIZATION_FILE"
    
    echo "✅ Updated with sed (backup created: kustomization.yaml.bak)"
fi

echo ""
echo "📋 Current image configuration:"
if command -v yq >/dev/null 2>&1; then
    yq eval '.images[0]' "$KUSTOMIZATION_FILE"
else
    grep -A2 "images:" "$KUSTOMIZATION_FILE"
fi

echo ""
echo "🚀 Ready to deploy! Run: oc apply -k ." 