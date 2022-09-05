/* vim: set filetype=yaml : */

package templates

var Node_1_24 = `
passwd:
  users:
    - name:          core
      password_hash: {{ .LoginPassword }}
      groups: [ rkt ]
{{- if .LoginPublicKey }}
      ssh_authorized_keys:
        - {{ .LoginPublicKey | quote }}
{{- end }}
  groups:
    - name: rkt
      system: true
systemd:
  units:
    - name: iptables-restore.service
      enable: true
    - name: ccloud-metadata-hostname.service
      enable: true
      contents: |
        [Unit]
        Description=Workaround for coreos-metadata hostname bug
        [Service]
        ExecStartPre=/usr/bin/curl -s http://169.254.169.254/latest/meta-data/hostname
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
    - name: kubelet.service
      enable: true
      contents: |
        [Unit]
        Description=Kubelet
        After=network-online.target nss-lookup.target
        Wants=network-online.target nss-lookup.target
        [Service]
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/run/kubelet-pod.uuid \
          --inherit-env \
          --net=host \
          --dns=host \
          --volume var-log,kind=host,source=/var/log \
          --mount volume=var-log,target=/var/log \
          --volume modprobe,kind=host,source=/usr/sbin/modprobe,readOnly=true  \
          --mount volume=modprobe,target=/usr/sbin/modprobe \
          --insecure-options=image"
        Environment="KUBELET_IMAGE_TAG={{ .KubeletImageTag }}"
        Environment="KUBELET_IMAGE_URL=docker://{{ .KubeletImage }}"
        Environment="KUBELET_IMAGE_ARGS=--name=kubelet --exec=/usr/local/bin/kubelet"
        ExecStartPre=/usr/bin/host identity-3.{{ .OpenstackRegion }}.cloud.sap
        ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
        ExecStartPre=-/opt/bin/rkt rm --uuid-file=/var/run/kubelet-pod.uuid
        ExecStart=/opt/bin/kubelet-wrapper \
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
        ExecStop=-/opt/bin/rkt stop --uuid-file=/var/run/kubelet-pod.uuid
        Restart=always
        RestartSec=10
        [Install]
        WantedBy=multi-user.target
    - name: wormhole.service
      contents: |
        [Unit]
        Description=Kubernikus Wormhole
        After=network-online.target nss-lookup.target
        Wants=network-online.target nss-lookup.target
        [Service]
        Slice=machine.slice
        ExecStartPre=/usr/bin/host identity-3.{{ .OpenstackRegion }}.cloud.sap
        ExecStartPre=/opt/bin/rkt fetch --insecure-options=image --pull-policy=new docker://{{ .KubernikusImage }}:{{ .KubernikusImageTag }}
        ExecStart=/opt/bin/rkt run \
          --inherit-env \
          --net=host \
          --dns=host \
          --volume var-lib-kubelet,kind=host,source=/var/lib/kubelet,readOnly=true \
          --mount volume=var-lib-kubelet,target=/var/lib/kubelet \
          --volume etc-kubernetes-certs,kind=host,source=/etc/kubernetes/certs,readOnly=true \
          --mount volume=etc-kubernetes-certs,target=/etc/kubernetes/certs \
          --insecure-options=image \
          --stage1-from-dir=stage1-fly.aci \
          docker://{{ .KubernikusImage }}:{{ .KubernikusImageTag }} \
          --name wormhole --exec wormhole -- client --listen {{ .ApiserverIP }}:{{ .ApiserverPort }} --kubeconfig=/var/lib/kubelet/kubeconfig
        ExecStopPost=/opt/bin/rkt gc --mark-only
        KillMode=mixed
        Restart=always
        RestartSec=10s
    - name: wormhole.path
      enable: true
      contents: |
        [Path]
        PathExists=/var/lib/kubelet/kubeconfig
        [Install]
        WantedBy=multi-user.target
    - name: kube-proxy.service
      enable: true
      contents: |
        [Unit]
        Description=Kube-Proxy
        After=network-online.target nss-lookup.target
        Wants=network-online.target nss-lookup.target
        [Service]
        Slice=machine.slice
        ExecStartPre=/usr/bin/host identity-3.{{ .OpenstackRegion }}.cloud.sap
        ExecStart=/opt/bin/rkt run \
          --trust-keys-from-https \
          --inherit-env \
          --net=host \
          --dns=host \
          --volume etc-kubernetes,kind=host,source=/etc/kubernetes,readOnly=true \
          --mount volume=etc-kubernetes,target=/etc/kubernetes \
          --volume lib-modules,kind=host,source=/lib/modules,readOnly=true \
          --mount volume=lib-modules,target=/lib/modules \
          --stage1-from-dir=stage1-fly.aci \
          --insecure-options=image \
          docker://{{ .KubeProxy }}:{{ .KubeProxyTag }} \
          --name kube-proxy \
          --exec /usr/local/bin/kube-proxy -- --config=/etc/kubernetes/kube-proxy/config
        ExecStopPost=/opt/bin/rkt gc --mark-only
        KillMode=mixed
        Restart=always
        RestartSec=10s
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
    - name: rkt-gc.service
      contents: |
        [Unit]
        Description=Garbage Collection for rkt
        [Service]
        Environment=GRACE_PERIOD=24h
        Type=oneshot
        ExecStart=/opt/bin/rkt gc --grace-period=${GRACE_PERIOD}
    - name: rkt-gc.timer
      enable: true
      command: start
      contents: |
        [Unit]
        Description=Periodic Garbage Collection for rkt
        [Timer]
        OnActiveSec=0s
        OnUnitActiveSec=12h
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
          [plugins."containerd.runtime.v1.linux"]
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
          MIIGTDCCBDSgAwIBAgIQXQPZPTFhXY9Iizlwx48bmTANBgkqhkiG9w0BAQsFADBO
          MQswCQYDVQQGEwJERTERMA8GA1UEBwwIV2FsbGRvcmYxDzANBgNVBAoMBlNBUCBB
          RzEbMBkGA1UEAwwSU0FQIEdsb2JhbCBSb290IENBMB4XDTEyMDQyNjE1NDE1NVoX
          DTMyMDQyNjE1NDYyN1owTjELMAkGA1UEBhMCREUxETAPBgNVBAcMCFdhbGxkb3Jm
          MQ8wDQYDVQQKDAZTQVAgQUcxGzAZBgNVBAMMElNBUCBHbG9iYWwgUm9vdCBDQTCC
          AiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAOrxJKFFA1eTrZg1Ux8ax6n/
          LQRHZlgLc2FZpfyAgwvkt71wLkPLiTOaRb3Bd1dyydpKcwJLy0dzGkunzNkPRSFz
          bKy2IPS0RS45hUCCPzhGnqQM6TcDYWeWpSUvygqujgb/cAG0mSJpvzAD3SMDQ+VJ
          Az5Ryq4IrP7LkfCb63LKZxLsHEkEcNKoGPsSsd4LTwuEIyM3ZHcCoA97m6hvgLWV
          GLzLIQMEblkswqX29z7JZH+zJopoqZB6eEogE2YpExkw52PufytEslDY3dyVubjp
          GlvD4T03F2zm6CYleMwgWbATLVYvk2I9WfqPAP+ln2IU9DZzegSMTWHCE+jizaiq
          b5f5s7m8f+cz7ndHSrz8KD/S9iNdWpuSlknHDrh+3lFTX/uWNBRs5mC/cdejcqS1
          v6erflyIfqPWWO6PxhIs49NL9Lix3ou6opJo+m8K757T5uP/rQ9KYALIXvl2uFP7
          0CqI+VGfossMlSXa1keagraW8qfplz6ffeSJQWO/+zifbfsf0tzUAC72zBuO0qvN
          E7rSbqAfpav/o010nKP132gbkb4uOkUfZwCuvZjA8ddsQ4udIBRj0hQlqnPLJOR1
          PImrAFC3PW3NgaDEo9QAJBEp5jEJmQghNvEsmzXgABebwLdI9u0VrDz4mSb6TYQC
          XTUaSnH3zvwAv8oMx7q7AgMBAAGjggEkMIIBIDAOBgNVHQ8BAf8EBAMCAQYwEgYD
          VR0TAQH/BAgwBgEB/wIBATAdBgNVHQ4EFgQUg8dB/Q4mTynBuHmOhnrhv7XXagMw
          gdoGA1UdIASB0jCBzzCBzAYKKwYBBAGFNgRkATCBvTAmBggrBgEFBQcCARYaaHR0
          cDovL3d3dy5wa2kuY28uc2FwLmNvbS8wgZIGCCsGAQUFBwICMIGFHoGCAEMAZQBy
          AHQAaQBmAGkAYwBhAHQAZQAgAFAAbwBsAGkAYwB5ACAAYQBuAGQAIABDAGUAcgB0
          AGkAZgBpAGMAYQB0AGkAbwBuACAAUAByAGEAYwB0AGkAYwBlACAAUwB0AGEAdABl
          AG0AZQBuAHQAIABvAGYAIABTAEEAUAAgAEEARzANBgkqhkiG9w0BAQsFAAOCAgEA
          0HpCIaC36me6ShB3oHDexA2a3UFcU149nZTABPKT+yUCnCQPzvK/6nJUc5I4xPfv
          2Q8cIlJjPNRoh9vNSF7OZGRmWQOFFrPWeqX5JA7HQPsRVURjJMeYgZWMpy4t1Tof
          lF13u6OY6xV6A5kQZIISFj/dOYLT3+O7wME5SItL+YsNh6BToNU0xAZt71Z8JNdY
          VJb2xSPMzn6bNXY8ioGzHlVxfEvzMqebV0KY7BTXR3y/Mh+v/RjXGmvZU6L/gnU7
          8mTRPgekYKY8JX2CXTqgfuW6QSnJ+88bHHMhMP7nPwv+YkPcsvCPBSY08ykzFATw
          SNoKP1/QFtERVUwrUXt3Cufz9huVysiy23dEyfAglgCCRWA+ZlaaXfieKkUWCJaE
          Kw/2Jqz02HDc7uXkFLS1BMYjr3WjShg1a+ulYvrBhNtseRoZT833SStlS/jzZ8Bi
          c1dt7UOiIZCGUIODfcZhO8l4mtjh034hdARLF0sUZhkVlosHPml5rlxh+qn8yJiJ
          GJ7CUQtNCDBVGksVlwew/+XnesITxrDjUMu+2297at7wjBwCnO93zr1/wsx1e2Um
          Xn+IfM6K/pbDar/y6uI9rHlyWu4iJ6cg7DAPJ2CCklw/YHJXhDHGwheO/qSrKtgz
          PGHZoN9jcvvvWDLUGtJkEotMgdFpEA2XWR83H4fVFVc=
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
    - path: /var/lib/iptables/rules-save
      filesystem: root
      mode: 0644
      contents:
        inline: |
          *nat
          :PREROUTING ACCEPT [0:0]
          :INPUT ACCEPT [0:0]
          :OUTPUT ACCEPT [0:0]
          :POSTROUTING ACCEPT [0:0]
          -A POSTROUTING -p tcp ! -d {{ .ClusterCIDR }} -m addrtype ! --dst-type LOCAL -j MASQUERADE --to-ports 32768-65535
          -A POSTROUTING -p udp ! -d {{ .ClusterCIDR }} -m addrtype ! --dst-type LOCAL -j MASQUERADE --to-ports 32768-65535
          -A POSTROUTING -p icmp ! -d {{ .ClusterCIDR }} -m addrtype ! --dst-type LOCAL -j MASQUERADE
          COMMIT
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
    - path: /etc/kubernetes/environment
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          NODE_NAME={{ .NodeName }}
    - path: /etc/kubernetes/certs/kubelet-clients-ca.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |-
{{ .KubeletClientsCA | indent 10 }}
    - path: /etc/kubernetes/certs/apiserver-clients-system-kube-proxy-key.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |-
{{ .ApiserverClientsSystemKubeProxyKey | indent 10 }}
    - path: /etc/kubernetes/certs/apiserver-clients-system-kube-proxy.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |-
{{ .ApiserverClientsSystemKubeProxy | indent 10 }}
    - path: /etc/kubernetes/certs/tls-ca.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |-
{{ .TLSCA | indent 10 }}
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
                 certificate-authority: /etc/kubernetes/certs/tls-ca.pem
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
    - path: /etc/kubernetes/kube-proxy/kubeconfig
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          apiVersion: v1
          kind: Config
          clusters:
            - name: local
              cluster:
                 certificate-authority: /etc/kubernetes/certs/tls-ca.pem
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
                client-certificate: /etc/kubernetes/certs/apiserver-clients-system-kube-proxy.pem
                client-key: /etc/kubernetes/certs/apiserver-clients-system-kube-proxy-key.pem
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
    - path: /etc/kubernetes/kube-proxy/config
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          apiVersion: kubeproxy.config.k8s.io/v1alpha1
          kind: KubeProxyConfiguration
          bindAddress: 0.0.0.0
          clientConnection:
            acceptContentTypes: ""
            burst: 10
            contentType: application/vnd.kubernetes.protobuf
            kubeconfig: "/etc/kubernetes/kube-proxy/kubeconfig"
            qps: 5
          clusterCIDR: "{{ .ClusterCIDR }}"
          configSyncPeriod: 15m0s
          conntrack:
            max: 0
            maxPerCore: 32768
            min: 131072
            tcpCloseWaitTimeout: 1h0m0s
            tcpEstablishedTimeout: 24h0m0s
          enableProfiling: false
          featureGates:
          healthzBindAddress: 0.0.0.0:10256
          hostnameOverride: {{ .NodeName }}
          iptables:
            masqueradeAll: false
            masqueradeBit: 14
            minSyncPeriod: 0s
            syncPeriod: 30s
          metricsBindAddress: 127.0.0.1:10249
          mode: "iptables"
          oomScoreAdj: -999
          portRange: ""
          resourceContainer: /kube-proxy
          udpTimeoutMilliseconds: 250ms
    - path: /etc/kubernetes/openstack/openstack.config
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          [Global]
          auth-url = {{ .OpenstackAuthURL }}
          username = {{ .OpenstackUsername }}
          password = {{ .OpenstackPassword }}
          domain-name = {{ .OpenstackDomain }}
          region = {{ .OpenstackRegion }}

          [LoadBalancer]
          lb-version=v2
          subnet-id = {{ .OpenstackLBSubnetID }}
          floating-network-id = {{ .OpenstackLBFloatingNetworkID }}
          create-monitor = yes
          monitor-delay = 1m
          monitor-timeout = 30s
          monitor-max-retries = 3

          [BlockStorage]
          trust-device-path = no

          [Route]
          router-id = {{ .OpenstackRouterID }}
    - path: /etc/coreos/update.conf
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          REBOOT_STRATEGY="off"
    - path: /opt/bin/rkt
      filesystem: root
      mode: 0755
      contents:
        remote:
          url: https://repo.{{.OpenstackRegion}}.cloud.sap/controlplane/flatcar-rkt/rkt-v1.30.0.gz
          compression: gzip
          verification:
            hash:
              function: sha512
              sum: 259fd4d1e1d33715c03ec1168af42962962cf70abc5ae9976cf439949f3bcdaf97110455fcf40c415a2adece28f6a52b46f8abd180cad1ee2e802d41a590b35f
    - path: /opt/rkt/stage1-fly.aci
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: https://repo.{{.OpenstackRegion}}.cloud.sap/controlplane/flatcar-rkt/stage1-fly-rkt-v1.30.0.aci
          verification:
            hash:
              function: sha512
              sum: 624bcf48b6829d2ac05c5744996d0fbbe2a0757bf2e5ad859f962a7001bb81980b0aa7be8532f3ec1ef7bbf025bbd089f5aa2eee9fdadefed1602343624750f1
    - path: /opt/rkt/stage1-coreos.aci
      filesystem: root
      mode: 0644
      contents:
        remote:
          url: https://repo.{{.OpenstackRegion}}.cloud.sap/controlplane/flatcar-rkt/stage1-coreos-rkt-v1.30.0.aci
          verification:
            hash:
              function: sha512
              sum: b295e35daab8ca312aeb516a59e79781fd8661d585ecd6c2714bbdec9738ee9012114a2ec886b19cb6eb2e212d72da6f902f02ca889394ef23dbd81fbf147f8c
    - path: /etc/rkt/paths.d/stage1.json
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          {
            "rktKind": "paths",
            "rktVersion": "v1",
            "stage1-images": "/opt/rkt"
          }

    - path: /opt/bin/kubelet-wrapper
      filesystem: root
      mode: 0755
      contents:
        inline: |-
          #!/bin/bash
          set -e
          function require_ev_all() {
            for rev in $@ ; do
              if [[ -z "${!rev}" ]]; then
                echo "${rev}" is not set
                exit 1
              fi
            done
          }
          function require_ev_one() {
            for rev in $@ ; do
              if [[ ! -z "${!rev}" ]]; then
                return
              fi
            done
            echo One of $@ must be set
            exit 1
          }
          if [[ -n "${KUBELET_VERSION}" ]]; then
            echo KUBELET_VERSION environment variable is deprecated, please use KUBELET_IMAGE_TAG instead
          fi
          if [[ -n "${KUBELET_ACI}" ]]; then
            echo KUBELET_ACI environment variable is deprecated, please use the KUBELET_IMAGE_URL instead
          fi
          if [[ -n "${RKT_OPTS}" ]]; then
            echo RKT_OPTS environment variable is deprecated, please use the RKT_RUN_ARGS instead
          fi
          KUBELET_IMAGE_TAG="${KUBELET_IMAGE_TAG:-${KUBELET_VERSION}}"
          require_ev_one KUBELET_IMAGE KUBELET_IMAGE_TAG
          KUBELET_IMAGE_URL="${KUBELET_IMAGE_URL:-${KUBELET_ACI:-docker://quay.io/coreos/hyperkube}}"
          KUBELET_IMAGE="${KUBELET_IMAGE:-${KUBELET_IMAGE_URL}:${KUBELET_IMAGE_TAG}}"
          RKT_RUN_ARGS="${RKT_RUN_ARGS} ${RKT_OPTS}"
          if [[ "${KUBELET_IMAGE%%/*}" == "quay.io" ]] && ! (echo "${RKT_RUN_ARGS}" | grep -q trust-keys-from-https); then
            RKT_RUN_ARGS="${RKT_RUN_ARGS} --trust-keys-from-https"
          elif [[ "${KUBELET_IMAGE%%/*}" == "docker:" ]] && ! (echo "${RKT_RUN_ARGS}" | grep -q insecure-options); then
            RKT_RUN_ARGS="${RKT_RUN_ARGS} --insecure-options=image"
          fi
          mkdir --parents /etc/kubernetes
          mkdir --parents /var/lib/containerd
          mkdir --parents /var/lib/kubelet
          mkdir --parents /run/kubelet
          RKT="${RKT:-/opt/bin/rkt}"
          RKT_STAGE1_ARG="${RKT_STAGE1_ARG:---stage1-from-dir=stage1-fly.aci}"
          KUBELET_IMAGE_ARGS=${KUBELET_IMAGE_ARGS:---exec=/kubelet}
          set -x
          exec ${RKT} ${RKT_GLOBAL_ARGS} \
            run ${RKT_RUN_ARGS} \
            --volume coreos-etc-kubernetes,kind=host,source=/etc/kubernetes,readOnly=false \
            --volume coreos-etc-ssl-certs,kind=host,source=/etc/ssl/certs,readOnly=true \
            --volume coreos-usr-share-certs,kind=host,source=/usr/share/ca-certificates,readOnly=true \
            --volume coreos-var-lib-containerd,kind=host,source=/var/lib/containerd,readOnly=false \
            --volume coreos-var-lib-kubelet,kind=host,source=/var/lib/kubelet,readOnly=false,recursive=true \
            --volume coreos-var-log,kind=host,source=/var/log,readOnly=false \
            --volume coreos-os-release,kind=host,source=/usr/lib/os-release,readOnly=true \
            --volume coreos-run,kind=host,source=/run,readOnly=false \
            --volume coreos-run-torcx-unpack,kind=host,source=/run/torcx/unpack,readOnly=false \
            --volume coreos-lib-modules,kind=host,source=/lib/modules,readOnly=true \
            --volume coreos-etc-machine-id,kind=host,source=/etc/machine-id,readOnly=true \
            --mount volume=coreos-etc-kubernetes,target=/etc/kubernetes \
            --mount volume=coreos-etc-ssl-certs,target=/etc/ssl/certs \
            --mount volume=coreos-usr-share-certs,target=/usr/share/ca-certificates \
            --mount volume=coreos-var-lib-containerd,target=/var/lib/containerd \
            --mount volume=coreos-var-lib-kubelet,target=/var/lib/kubelet \
            --mount volume=coreos-var-log,target=/var/log \
            --mount volume=coreos-os-release,target=/etc/os-release \
            --mount volume=coreos-run-torcx-unpack,target=/run/torcx/unpack \
            --mount volume=coreos-run,target=/run \
            --mount volume=coreos-lib-modules,target=/lib/modules \
            --mount volume=coreos-etc-machine-id,target=/etc/machine-id \
            --hosts-entry host \
            ${RKT_STAGE1_ARG} \
            ${KUBELET_IMAGE} \
              ${KUBELET_IMAGE_ARGS} \
              -- "$@"
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
