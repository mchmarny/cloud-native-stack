#!/usr/bin/env bash
# GPU Operator Uninstallation Script
# Generated from Cloud Native Stack Recipe
# Timestamp: 2026-01-02T18:01:22Z
# Bundler Version: 0.8.12
# Recipe Bundler Version: 0.8.12

set -euo pipefail

NAMESPACE="gpu-operator"

log_info() {
    echo "[INFO] $1"
}

log_error() {
    echo "[ERROR] $1"
}

uninstall_gpu_operator() {
    log_info "Uninstalling GPU Operator..."
    
    if helm list -n "$NAMESPACE" | grep -q gpu-operator; then
        helm uninstall gpu-operator -n "$NAMESPACE" --wait
        log_info "GPU Operator uninstalled."
    else
        log_info "GPU Operator not found."
    fi
}

delete_crds() {
    log_info "Deleting CRDs..."
    kubectl delete crd clusterpolicies.nvidia.com --ignore-not-found=true
    log_info "CRDs deleted."
}

delete_namespace() {
    log_info "Deleting namespace: $NAMESPACE"
    kubectl delete namespace "$NAMESPACE" --ignore-not-found=true --wait
    log_info "Namespace deleted."
}

main() {
    log_info "Starting GPU Operator uninstallation..."
    
    uninstall_gpu_operator
    delete_crds
    
    read -p "Delete namespace $NAMESPACE? (y/N): " confirm
    if [[ "$confirm" == "y" || "$confirm" == "Y" ]]; then
        delete_namespace
    fi
    
    log_info "GPU Operator uninstallation complete!"
}

main "$@"
