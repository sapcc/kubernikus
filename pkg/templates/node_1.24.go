/* vim: set filetype=yaml : */

package templates

var Node_1_24 = `
variant: flatcar
version: 1.0.0
kernel_arguments:
  should_not_exist:
    - flatcar.autologin
passwd:
  users:
    - name:          core
      password_hash: {{ .LoginPassword }}
{{- if .LoginPublicKey }}
      ssh_authorized_keys:
        - {{ .LoginPublicKey | quote }}
{{- end }}
systemd:
  units:
    - name: ccloud-metadata-hostname.service
      enabled: true
      contents: |
        [Unit]
        Description=Workaround for coreos-metadata hostname bug
        [Service]
        ExecStartPre=/usr/bin/curl -sf http://169.254.169.254/latest/meta-data/hostname
        ExecStartPre=/usr/bin/bash -c "/usr/bin/systemctl set-environment COREOS_OPENSTACK_HOSTNAME=$(curl -s http://169.254.169.254/latest/meta-data/hostname)"
        ExecStart=/usr/bin/hostnamectl set-hostname ${COREOS_OPENSTACK_HOSTNAME}
        Restart=on-failure
        RestartSec=5
        RemainAfterExit=yes
        [Install]
        WantedBy=multi-user.target
    - name: containerd.service
      enabled: true
      dropins:
        - name: 10-custom-config.conf
          contents: |
            [Service]
            ExecStart=
            ExecStart=/usr/bin/containerd
    - name: docker.service
      enabled: true
      dropins:
        - name: 20-docker-opts.conf
          contents: |
            [Service]
            Environment="DOCKER_OPTS=--iptables=false --bridge=none"
    - name: kubelet.service
      enabled: true
      contents: |
        [Unit]
        Description=Kubelet
        After=network-online.target nss-lookup.target containerd.service
        Wants=network-online.target nss-lookup.target
        [Service]
        Restart=always
        RestartSec=2s
        StartLimitInterval=0
        KillMode=process
        User=root
        CPUAccounting=true
        MemoryAccounting=true
        ExecStartPre=docker run --rm -v /opt/bin:/opt/bin {{ .KubeletImage }}:{{ .KubeletImageTag }} cp --preserve=mode /usr/local/bin/kubelet /opt/bin/kubelet
        ExecStart=/opt/bin/kubelet \
          --cert-dir=/var/lib/kubelet/pki \
          --container-runtime=remote \
          --container-runtime-endpoint=unix:///run/containerd/containerd.sock \
          --cloud-provider=external \
          --config=/etc/kubernetes/kubelet/config \
          --bootstrap-kubeconfig=/etc/kubernetes/bootstrap/kubeconfig \
          --hostname-override={{ .NodeName }} \
          --kubeconfig=/var/lib/kubelet/kubeconfig \
          --lock-file=/var/run/lock/kubelet.lock \
          --pod-infra-container-image={{ .PauseImage }}:{{ .PauseImageTag }} \
          --node-labels=kubernikus.cloud.sap/cni=true{{ if .NodeLabels }},{{ .NodeLabels | join "," }}{{ end }} \
{{- if .NodeTaints }}
          --register-with-taints={{ .NodeTaints | join "," }} \
{{- end }}
          --volume-plugin-dir=/var/lib/kubelet/volumeplugins \
          --rotate-certificates \
          --exit-on-lock-contention

        [Install]
        WantedBy=multi-user.target
    - name: updatecertificates.service
      command: start
      enabled: true
      contents: |
        [Unit]
        Description=Update the certificates w/ self-signed root CAs
        ConditionPathIsSymbolicLink=!/etc/ssl/certs/381107d7.0
        Before=early-docker.service docker.service
        [Service]
        ExecStart=/usr/sbin/update-ca-certificates
        RemainAfterExit=yes
        Type=oneshot
        [Install]
        WantedBy=multi-user.target
    - name: containerd-config-replace.service
      enabled: true
      contents: |
        [Unit]
        Description=Modify startup configuration file of containerd
        Before=containerd.service
        After=network-online.target
        Wants=network-online.target
        [Service]
        Type=oneshot
        WorkingDirectory=/opt/bin/
        ExecStartPre=/bin/sh -c 'until host repo.{{ .OpenstackRegion }}.cloud.sap; do sleep 1; done'
        ExecStart=/opt/bin/containerd-config-replace.sh
        [Install]
        WantedBy=multi-user.target
storage:
  files:
    - path: /etc/crictl.yaml
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          runtime-endpoint: unix:///run/containerd/containerd.sock
    - path: /etc/profile.d/envs.sh
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          export CONTAINERD_NAMESPACE=k8s.io
    - path: /etc/systemd/resolved.conf
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          [Resolve]
          DNSStubListener=no
    - path: /etc/systemd/network/50-kubernikus.netdev
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          [NetDev]
          Description=Kubernikus Dummy Interface
          Name=kubernikus
          Kind=dummy
    - path: /etc/systemd/network/51-kubernikus.network
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          [Match]
          Name=kubernikus
          [Network]
          DHCP=no
          Address={{ .ApiserverIP }}/32
    - path: /etc/udev/rules.d/99-vmware-scsi-udev.rules
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          ACTION=="add", SUBSYSTEMS=="scsi", ATTRS{vendor}=="VMware  ", ATTRS{model}=="Virtual disk", RUN+="/bin/sh -c 'echo 180 >/sys$DEVPATH/timeout'"
    - path: /etc/ssl/certs/SAPGlobalRootCA.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          -----BEGIN CERTIFICATE-----
          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
          -----END CERTIFICATE-----
    - path: /etc/ssl/certs/SAPNetCA_G2.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          -----BEGIN CERTIFICATE-----
          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
          OfI=
          -----END CERTIFICATE-----
    - path: /etc/sysctl.d/10-enable-icmp-redirects.conf
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          net.ipv4.conf.all.accept_redirects=1
    - path: /etc/sysctl.d/20-inotify-max-user.conf
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          fs.inotify.max_user_instances=8192
          fs.inotify.max_user_watches=524288
    - path: /etc/kubernetes/certs/kubelet-clients-ca.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
{{ .KubeletClientsCA | indent 10 }}
    - path: /etc/kubernetes/bootstrap/kubeconfig
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          apiVersion: v1
          kind: Config
          clusters:
            - name: local
              cluster:
                 certificate-authority-data: {{ .TLSCA | b64enc }}
                 server: {{ .ApiserverURL }}
          contexts:
            - name: local
              context:
                cluster: local
                user: local
          current-context: local
          users:
            - name: local
              user:
                token: {{ .BootstrapToken }}
    - path: /etc/kubernetes/kubelet/config
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          kind: KubeletConfiguration
          apiVersion: kubelet.config.k8s.io/v1beta1
          readOnlyPort: 0
          clusterDomain: {{ .ClusterDomain }}
          clusterDNS: [{{ .ClusterDNSAddress }}]
          authentication:
            x509:
              clientCAFile: /etc/kubernetes/certs/kubelet-clients-ca.pem
            anonymous:
              enabled: true
          rotateCertificates: true
          nodeLeaseDurationSeconds: 20
          cgroupDriver: systemd
          featureGates:
{{- if not .NoCloud }}
            CSIMigration: true
            CSIMigrationOpenStack: true
            ExpandCSIVolumes: true
{{- end }}
    - path: /etc/flatcar/update.conf
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          REBOOT_STRATEGY="off"
    - path: /etc/modules-load.d/br_netfilter.conf
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: br_netfilter
    - path: /etc/sysctl.d/30-br_netfilter.conf
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |
          net.bridge.bridge-nf-call-ip6tables = 1
          net.bridge.bridge-nf-call-iptables = 1
    - path: /opt/bin/containerd-config-replace.sh
      filesystem: root
      mode: 0755
      overwrite: true
      contents:
        inline: |
          #!/usr/bin/env bash
          set -eux
          mkdir -p /etc/containerd
          # copy original file just in case config injection fails
          if [ -f "/run/torcx/unpack/docker/usr/share/containerd/config.toml" ]; then
            cp /run/torcx/unpack/docker/usr/share/containerd/config.toml /etc/containerd/config.toml
          fi
          # without torcx
          if [ -f "/usr/share/containerd/config.toml" ]; then
            cp -f /usr/share/containerd/config.toml /etc/containerd/config.toml
          fi
          cp -f /etc/containerd/config.toml output.toml
          #download xtoml
          curl https://repo.{{ .OpenstackRegion }}.cloud.sap/controlplane/xtoml/xtoml -o xtoml
          chmod +x xtoml
          ./xtoml add --file output.toml --plugin "io.containerd.grpc.v1.cri" --key sandbox_image --value "{{ .PauseImage }}:{{ .PauseImageTag }}" --type string
          ./xtoml add --file output.toml --plugin "io.containerd.grpc.v1.cri" --key enable_unprivileged_ports --value true --type bool
          ./xtoml add --file output.toml --plugin "io.containerd.grpc.v1.cri" --key enable_unprivileged_icmp --value true --type bool
          cp output.toml /etc/containerd/config.toml
`
