#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VM_DIR="$SCRIPT_DIR"
DISK="$VM_DIR/vyos.qcow2"
ISO="$VM_DIR/vyos.iso"
SEED_ISO="$VM_DIR/seed.iso"
CLOUD_INIT_DIR="$VM_DIR/cloud-init"

SSH_PORT="${VYOS_SSH_PORT:-2222}"
API_PORT="${VYOS_API_PORT:-8443}"
MEMORY="${VYOS_MEMORY:-2G}"

usage() {
    cat <<EOF
Usage: $0 <command>

Commands:
  download    Download latest VyOS rolling nightly ISO
  seed        Generate cloud-init seed ISO (requires xorriso)
  install     Boot live ISO for VyOS installation to disk (interactive)
  start       Boot VyOS from disk with cloud-init
  status      Check if VM API is reachable
  help        Show this help

Environment variables:
  VYOS_SSH_PORT   SSH forward port (default: 2222)
  VYOS_API_PORT   API forward port (default: 8443)
  VYOS_MEMORY     VM memory (default: 2G)

After first run:
  1. $0 download
  2. $0 seed
  3. $0 install    (interactive: run 'install image' inside VyOS, then 'poweroff')
  4. $0 start      (boots from disk with cloud-init provisioning)

Test connectivity:
  ssh -p $SSH_PORT vyos@localhost
  curl -k https://localhost:$API_PORT/retrieve \\
    --form data='{"op":"showConfig","path":[]}' \\
    --form key='integration-test-key'
EOF
}

cmd_download() {
    echo "Fetching latest VyOS rolling nightly ISO URL..."
    local iso_url
    iso_url=$(curl -sL "https://api.github.com/repos/vyos/vyos-rolling-nightly-builds/releases/latest" \
        | jq -r '.assets[] | select(.name | endswith("-generic-amd64.iso")) | .browser_download_url')

    if [[ -z "$iso_url" || "$iso_url" == "null" ]]; then
        echo "ERROR: Could not find VyOS ISO download URL" >&2
        exit 1
    fi

    echo "Downloading: $iso_url"
    curl -L -o "$ISO" "$iso_url"
    echo "Downloaded to $ISO"
}

cmd_seed() {
    if ! command -v xorriso &>/dev/null; then
        echo "ERROR: xorriso not found. Install with: sudo dnf install xorriso" >&2
        exit 1
    fi

    echo "Generating cloud-init seed ISO..."
    xorriso -as mkisofs -joliet -rock -volid "cidata" \
        -o "$SEED_ISO" \
        "$CLOUD_INIT_DIR/meta-data" \
        "$CLOUD_INIT_DIR/user-data"
    echo "Created $SEED_ISO"
}

cmd_install() {
    if [[ ! -f "$ISO" ]]; then
        echo "ERROR: VyOS ISO not found at $ISO" >&2
        echo "Run: $0 download" >&2
        exit 1
    fi

    if [[ ! -f "$DISK" ]]; then
        echo "Creating disk image..."
        qemu-img create -f qcow2 "$DISK" 4G
    fi

    echo "Booting live ISO for installation..."
    echo "Inside VyOS: run 'install image', follow prompts, then 'poweroff'"
    echo ""
    qemu-system-x86_64 -enable-kvm -m "$MEMORY" -smp 2 \
        -boot d -cdrom "$ISO" \
        -drive "file=$DISK,if=virtio" \
        -netdev "user,id=net0,hostfwd=tcp::${SSH_PORT}-:22,hostfwd=tcp::${API_PORT}-:443" \
        -device e1000e,netdev=net0 \
        -serial mon:stdio -nographic
}

cmd_start() {
    if [[ ! -f "$DISK" ]]; then
        echo "ERROR: Disk image not found at $DISK" >&2
        echo "Run: $0 install" >&2
        exit 1
    fi

    local seed_args=()
    if [[ -f "$SEED_ISO" ]]; then
        seed_args=(-drive "file=$SEED_ISO,format=raw,if=virtio")
    else
        echo "WARNING: No seed ISO found. Cloud-init will not run." >&2
        echo "Run: $0 seed" >&2
    fi

    echo "Starting VyOS VM..."
    echo "  SSH:  ssh -p $SSH_PORT vyos@localhost"
    echo "  API:  https://localhost:$API_PORT"
    echo ""
    qemu-system-x86_64 -enable-kvm -m "$MEMORY" -smp 2 \
        -drive "file=$DISK,if=virtio" \
        "${seed_args[@]}" \
        -netdev "user,id=net0,hostfwd=tcp::${SSH_PORT}-:22,hostfwd=tcp::${API_PORT}-:443" \
        -device e1000e,netdev=net0 \
        -serial mon:stdio -nographic
}

cmd_status() {
    echo "Checking VyOS API at https://localhost:$API_PORT ..."
    if curl -sk --max-time 5 "https://localhost:$API_PORT/retrieve" \
        --form 'data={"op":"showConfig","path":[]}' \
        --form 'key=integration-test-key' \
        -o /dev/null -w "%{http_code}" 2>/dev/null | grep -q "200"; then
        echo "VyOS API is reachable."
    else
        echo "VyOS API is not reachable."
        exit 1
    fi
}

case "${1:-help}" in
    download) cmd_download ;;
    seed)     cmd_seed ;;
    install)  cmd_install ;;
    start)    cmd_start ;;
    status)   cmd_status ;;
    help|*)   usage ;;
esac
