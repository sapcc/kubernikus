/* vim: set filetype=yaml : */

package templates

var Node_1_24 = `
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
      enable: true
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
      enable: true
      dropins:
        - name: 10-custom-config.conf
          contents: |
            [Service]
            ExecStart=
            ExecStart=/usr/bin/env PATH=${TORCX_BINDIR}:${PATH} ${TORCX_BINDIR}/containerd
    - name: docker.service
      enable: true
      dropins:
        - name: 20-docker-opts.conf
          contents: |
            [Service]
            Environment="DOCKER_OPTS=--iptables=false --bridge=none"
    - name: kubelet.service
      enable: true
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
      enable: true
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
storage:
  files:
    - path: /etc/crictl.yaml
      filesystem: root
      mode: 0644
      contents:
        inline: |
          runtime-endpoint: unix:///run/containerd/containerd.sock
    - path: /etc/profile.d/envs.sh
      filesystem: root
      mode: 0644
      contents:
        inline: |
          export CONTAINERD_NAMESPACE=k8s.io
      #copied from /run/torcx/unpack/docker/usr/share/containerd/config.toml
    - path: /etc/containerd/config.toml
      filesystem: root
      mode: 0644
      contents:
        inline: |
          version = 2
          # persistent data location
          root = "/var/lib/containerd"
          # runtime state information
          state = "/run/containerd"
          # set containerd as a subreaper on linux when it is not running as PID 1
          subreaper = true
          # set containerd's OOM score
          oom_score = -999
          disabled_plugins = []
          # grpc configuration
          [grpc]
          address = "/run/containerd/containerd.sock"
          # socket uid
          uid = 0
          # socket gid
          gid = 0
          [plugins."io.containerd.runtime.v1.linux"]
          # shim binary name/path
          shim = "containerd-shim"
          # runtime binary name/path
          runtime = "runc"
          # do not use a shim when starting containers, saves on memory but
          # live restore is not supported
          no_shim = false
          [plugins."io.containerd.grpc.v1.cri"]
          # enable SELinux labeling
          enable_selinux = true
          sandbox_image = "{{ .PauseImage }}:{{ .PauseImageTag }}"
          # compat with previous docker based runtime
          enable_unprivileged_ports = true
          enable_unprivileged_icmp = true
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc]
          # setting runc.options unsets parent settings
          runtime_type = "io.containerd.runc.v2"
          [plugins."io.containerd.grpc.v1.cri".containerd.runtimes.runc.options]
          SystemdCgroup = true
    - path: /etc/systemd/resolved.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |
          [Resolve]
          DNSStubListener=no
    - path: /etc/systemd/network/50-kubernikus.netdev
      filesystem: root
      mode: 0644
      contents:
        inline: |
          [NetDev]
          Description=Kubernikus Dummy Interface
          Name=kubernikus
          Kind=dummy
    - path: /etc/systemd/network/51-kubernikus.network
      filesystem: root
      mode: 0644
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
      contents:
        inline: |
          ACTION=="add", SUBSYSTEMS=="scsi", ATTRS{vendor}=="VMware  ", ATTRS{model}=="Virtual disk", RUN+="/bin/sh -c 'echo 180 >/sys$DEVPATH/timeout'"
    - path: /etc/ssl/certs/SAPGlobalRootCA.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |
          -----BEGIN CERTIFICATE-----
          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
          -----END CERTIFICATE-----
    - path: /etc/ssl/certs/SAPNetCA_G2.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |
          -----BEGIN CERTIFICATE-----
          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
          7HJNBB1ZTjnrxJAgCQgNBIQ=
          -----END CERTIFICATE-----
    - path: /etc/sysctl.d/10-enable-icmp-redirects.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          net.ipv4.conf.all.accept_redirects=1
    - path: /etc/sysctl.d/20-inotify-max-user.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          fs.inotify.max_user_instances=8192
          fs.inotify.max_user_watches=524288
    - path: /etc/kubernetes/certs/kubelet-clients-ca.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |-
{{ .KubeletClientsCA | indent 10 }}
    - path: /etc/kubernetes/bootstrap/kubeconfig
      filesystem: root
      mode: 0644
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
      contents:
        inline: |-
          REBOOT_STRATEGY="off"
    - path: /etc/modules-load.d/br_netfilter.conf
      filesystem: root
      mode: 0644
      contents:
        inline: br_netfilter
    - path: /etc/sysctl.d/30-br_netfilter.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |
          net.bridge.bridge-nf-call-ip6tables = 1
          net.bridge.bridge-nf-call-iptables = 1
`
