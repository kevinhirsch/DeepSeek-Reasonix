package cli

import (
	"flag"
	"fmt"
	"os"
	"strings"
)

func remoteCommand(args []string) int {
	if len(args) == 0 {
		remoteUsage()
		return 2
	}
	switch args[0] {
	case "bootstrap":
		return remoteBootstrapCommand(args[1:])
	default:
		remoteUsage()
		return 2
	}
}

func remoteBootstrapCommand(args []string) int {
	fs := flag.NewFlagSet("remote bootstrap", flag.ContinueOnError)
	name := fs.String("name", "", "Worker name (required)")
	output := fs.String("output", "", "Output script file (default: stdout)")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*name) == "" {
		fmt.Fprintln(os.Stderr, "Worker name is required (--name my-worker)")
		return 2
	}
	script := generateBootstrapScript(*name)
	if *output != "" {
		if err := os.WriteFile(*output, []byte(script), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to write script: %v\n", err)
			return 1
		}
		fmt.Printf("Bootstrap script written to %s\n", *output)
		return 0
	}
	fmt.Print(script)
	return 0
}

func remoteUsage() {
	fmt.Println("Usage: reasonix remote <subcommand>")
	fmt.Println("  bootstrap --name <name>  Generate worker bootstrap script")
	fmt.Println("  list                      List configured remotes")
}

func generateBootstrapScript(workerName string) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("# reasonix remote bootstrap — generated script for worker: " + workerName + "\n")
	b.WriteString("set -euo pipefail\n\n")

	b.WriteString("# ---- 1. Detect OS ----\n")
	b.WriteString("detect_os() {\n")
	b.WriteString("  if [ -f /etc/os-release ]; then\n")
	b.WriteString("    . /etc/os-release\n")
	b.WriteString("    echo \"$ID $VERSION_ID\"\n")
	b.WriteString("  elif [ -f /etc/debian_version ]; then\n")
	b.WriteString("    echo \"debian $(cat /etc/debian_version)\"\n")
	b.WriteString("  elif [ -f /etc/fedora-release ]; then\n")
	b.WriteString("    echo \"fedora $(rpm -q --qf '%{VERSION}' fedora-release)\"\n")
	b.WriteString("  else\n")
	b.WriteString("    echo \"unknown\"\n")
	b.WriteString("  fi\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 2. Install Docker if missing ----\n")
	b.WriteString("install_docker() {\n")
	b.WriteString("  if command -v docker &>/dev/null; then\n")
	b.WriteString("    echo \"Docker already installed: $(docker --version)\"\n")
	b.WriteString("    return 0\n")
	b.WriteString("  fi\n")
	b.WriteString("  echo \"Installing Docker...\"\n")
	b.WriteString("  OS_ID=\"$(detect_os | cut -d' ' -f1)\"\n")
	b.WriteString("  case \"$OS_ID\" in\n")
	b.WriteString("    ubuntu|debian)\n")
	b.WriteString("      curl -fsSL https://get.docker.com | sh\n")
	b.WriteString("      ;;\n")
	b.WriteString("    fedora|centos|rhel)\n")
	b.WriteString("      dnf -y install docker\n")
	b.WriteString("      systemctl enable --now docker\n")
	b.WriteString("      ;;\n")
	b.WriteString("    *)\n")
	b.WriteString("      echo \"Unsupported OS for automatic Docker install. Install Docker manually.\"\n")
	b.WriteString("      exit 1\n")
	b.WriteString("      ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 3. Probe system resources ----\n")
	b.WriteString("probe_system() {\n")
	b.WriteString("  CPU=$(nproc)\n")
	b.WriteString("  RAM_KB=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo 0)\n")
	b.WriteString("  DISK_KB=$(df --output=avail / 2>/dev/null | tail -1 || echo 0)\n")
	b.WriteString("  RAM_MB=$((RAM_KB / 1024))\n")
	b.WriteString("  DISK_MB=$((DISK_KB / 1024))\n")
	b.WriteString("  echo \"CPU: $CPU cores, RAM: $RAM_MB MB, Disk: $DISK_MB MB free\"\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 4. Calculate safe limits (80% of system) ----\n")
	b.WriteString("calc_limits() {\n")
	b.WriteString("  CPU_MAX=$(( $(nproc) * 80 / 100 ))\n")
	b.WriteString("  [ $CPU_MAX -lt 1 ] && CPU_MAX=1\n")
	b.WriteString("  RAM_KB=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo 0)\n")
	b.WriteString("  RAM_MAX_MB=$(( RAM_KB / 1024 * 80 / 100 ))\n")
	b.WriteString("  [ $RAM_MAX_MB -lt 256 ] && RAM_MAX_MB=256\n")
	b.WriteString("  echo \"Limits: CPU=$CPU_MAX, RAM=${RAM_MAX_MB}MB\"\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 5. Download reasonix-worker binary ----\n")
	b.WriteString("download_worker() {\n")
	b.WriteString("  ARCH=$(uname -m)\n")
	b.WriteString("  case \"$ARCH\" in x86_64) GOARCH=amd64;; aarch64) GOARCH=arm64;; *) echo \"Unsupported arch: $ARCH\"; exit 1;; esac\n")
	b.WriteString("  URL=\"https://github.com/kevinhirsch/DeepSeek-Reasonix/releases/latest/download/reasonix-worker-linux-$GOARCH\"\n")
	b.WriteString("  echo \"Downloading reasonix-worker for $GOARCH...\"\n")
	b.WriteString("  curl -fsSL \"$URL\" -o /usr/local/bin/reasonix-worker || {\n")
	b.WriteString("    echo \"Download failed. Build from source: go build ./cmd/reasonix-worker\"\n")
	b.WriteString("    exit 1\n")
	b.WriteString("  }\n")
	b.WriteString("  chmod +x /usr/local/bin/reasonix-worker\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 6. Write worker.toml ----\n")
	b.WriteString("write_config() {\n")
	b.WriteString("  CPU_MAX=$(( $(nproc) * 80 / 100 ))\n")
	b.WriteString("  [ $CPU_MAX -lt 1 ] && CPU_MAX=1\n")
	b.WriteString("  RAM_KB=$(grep MemTotal /proc/meminfo 2>/dev/null | awk '{print $2}' || echo 0)\n")
	b.WriteString("  RAM_MAX_MB=$(( RAM_KB / 1024 * 80 / 100 ))\n\n")
	b.WriteString("  cat > /etc/reasonix/worker.toml << EOF\n")
	b.WriteString("name = \"" + workerName + "\"\n")
	b.WriteString("server_url = \"http://localhost:9090\"\n")
	b.WriteString("auth_token = \"\\${REASONIX_REMOTE_TOKEN}\"\n")
	b.WriteString("max_concurrent = $CPU_MAX\n")
	b.WriteString("memory_limit_mb = $RAM_MAX_MB\n")
	b.WriteString("timeout_minutes = 30\n")
	b.WriteString("sandbox = \"docker\"\n")
	b.WriteString("EOF\n")
	b.WriteString("  echo \"Config written to /etc/reasonix/worker.toml\"\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 7. Prompt for API keys ----\n")
	b.WriteString("prompt_keys() {\n")
	b.WriteString("  echo \"Enter the REASONIX_REMOTE_TOKEN for connecting to the server:\"\n")
	b.WriteString("  read -s REMOTE_TOKEN\n")
	b.WriteString("  echo \"\"\n")
	b.WriteString("  if [ -n \"$REMOTE_TOKEN\" ]; then\n")
	b.WriteString("    export REASONIX_REMOTE_TOKEN=\"$REMOTE_TOKEN\"\n")
	b.WriteString("    echo \"Token set.\"\n")
	b.WriteString("  fi\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 8. Install systemd service with security hardening ----\n")
	b.WriteString("install_service() {\n")
	b.WriteString("  cat > /etc/systemd/system/reasonix-worker.service << 'SERVICEOF'\n")
	b.WriteString("[Unit]\n")
	b.WriteString("Description=reasonix Remote Worker\n")
	b.WriteString("After=network-online.target docker.service\n")
	b.WriteString("Wants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	b.WriteString("ExecStart=/usr/local/bin/reasonix-worker --config /etc/reasonix/worker.toml\n")
	b.WriteString("Restart=on-failure\n")
	b.WriteString("RestartSec=10\n")
	b.WriteString("EnvironmentFile=-/etc/reasonix/worker.env\n")
	b.WriteString("NoNewPrivileges=yes\n")
	b.WriteString("ProtectSystem=strict\n")
	b.WriteString("ProtectHome=yes\n")
	b.WriteString("ReadWritePaths=/var/lib/reasonix /tmp\n")
	b.WriteString("PrivateTmp=yes\n")
	b.WriteString("PrivateDevices=no\n")
	b.WriteString("ProtectKernelTunables=yes\n")
	b.WriteString("ProtectKernelModules=yes\n")
	b.WriteString("ProtectControlGroups=yes\n")
	b.WriteString("MemoryDenyWriteExecute=no\n")
	b.WriteString("RestrictRealtime=yes\n")
	b.WriteString("RestrictNamespaces=yes\n")
	b.WriteString("LockPersonality=yes\n")
	b.WriteString("CapabilityBoundingSet=CAP_SYS_ADMIN\n")
	b.WriteString("LimitNOFILE=65536\n")
	b.WriteString("SERVICEOF\n\n")
	b.WriteString("  systemctl daemon-reload\n")
	b.WriteString("  echo \"Systemd service installed.\"\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- 9. Start worker ----\n")
	b.WriteString("start_worker() {\n")
	b.WriteString("  systemctl enable --now reasonix-worker\n")
	b.WriteString("  echo \"Worker started. Check status: systemctl status reasonix-worker\"\n")
	b.WriteString("}\n\n")

	b.WriteString("# ---- Main ----\n")
	b.WriteString("echo \"reasonix remote bootstrap for worker: " + workerName + "\"\n")
	b.WriteString("echo \"\"\n")
	b.WriteString("detect_os\n")
	b.WriteString("install_docker\n")
	b.WriteString("probe_system\n")
	b.WriteString("calc_limits\n")
	b.WriteString("download_worker\n")
	b.WriteString("mkdir -p /etc/reasonix /var/lib/reasonix\n")
	b.WriteString("write_config\n")
	b.WriteString("prompt_keys\n")
	b.WriteString("install_service\n")
	b.WriteString("start_worker\n")
	b.WriteString("echo \"\"\n")
	b.WriteString("echo \"✓ Worker " + workerName + " bootstrap complete.\"\n")

	return b.String()
}
