/* vim: set filetype=yaml : */

package templates

var Node_1_14 = `
variant: flatcar
version: 1.0.0
kernel_arguments:
  should_exist:
    - systemd.unified_cgroup_hierarchy=0
    - systemd.legacy_systemd_cgroup_controller
  should_not_exist:
    - flatcar.autologin
passwd:
  users:
    - name:          core
      password_hash: {{ .LoginPassword }}
      groups: [rkt]
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
      enabled: true
    - name: ccloud-metadata-hostname.service
      enabled: true
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
    - name: docker.service
      enabled: true
      dropins:
        - name: 20-docker-opts.conf
          contents: |
            [Service]
            Environment="DOCKER_OPTS=--log-opt max-size=5m --log-opt max-file=5 --ip-masq=false --iptables=false --bridge=none"
            Environment="DOCKER_CGROUPS=--exec-opt native.cgroupdriver=cgroupfs"
    - name: flanneld.service
      enabled: true
      contents: |
        [Unit]
        Description=flannel - Network fabric for containers (System Application Container)
        Documentation=https://github.com/coreos/flannel
        After=etcd.service etcd2.service etcd-member.service network-online.target nss-lookup.target
        Requires=flannel-docker-opts.service
        Wants=network-online.target nss-lookup.target
        [Service]
        Type=notify
        Restart=always
        RestartSec=10s
        TimeoutStartSec=300
        LimitNOFILE=40000
        LimitNPROC=1048576
        Environment="FLANNEL_IMAGE_URL=docker://{{ .FlannelImage }}"
        Environment="FLANNEL_IMAGE_TAG={{ .FlannelImageTag }}"
        Environment="FLANNEL_OPTS=--ip-masq=true"
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/lib/flatcar/flannel-wrapper.uuid"
        EnvironmentFile=-/run/flannel/options.env
        ExecStartPre=/usr/bin/host identity-3.{{ .OpenstackRegion }}.cloud.sap
        ExecStartPre=/sbin/modprobe ip_tables
        ExecStartPre=/usr/bin/mkdir --parents /var/lib/flatcar /run/flannel
        ExecStartPre=-/opt/bin/rkt rm --uuid-file=/var/lib/flatcar/flannel-wrapper.uuid
        ExecStart=/opt/bin/flannel-wrapper $FLANNEL_OPTS
        ExecStop=-/opt/bin/rkt stop --uuid-file=/var/lib/flatcar/flannel-wrapper.uuid
        [Install]
        WantedBy=multi-user.target
      dropins:
        - name: 10-ccloud-opts.conf
          contents: |
            [Service]
            EnvironmentFile=/etc/kubernetes/environment
            Environment="FLANNEL_OPTS=-ip-masq=false \
                                      -kube-subnet-mgr=true \
                                      -kubeconfig-file=/var/lib/kubelet/kubeconfig \
                                      -kube-api-url={{ .ApiserverURL }}"
            Environment="RKT_RUN_ARGS=--uuid-file-save=/var/lib/flatcar/flannel-wrapper.uuid \
                                      --volume var-lib-kubelet,kind=host,source=/var/lib/kubelet,readOnly=true \
                                      --mount volume=var-lib-kubelet,target=/var/lib/kubelet \
                                      --volume etc-kubernetes-certs,kind=host,source=/etc/kubernetes/certs,readOnly=true \
                                      --mount volume=etc-kubernetes-certs,target=/etc/kubernetes/certs \
                                      --volume etc-kube-flannel,kind=host,source=/etc/kube-flannel,readOnly=true \
                                      --mount volume=etc-kube-flannel,target=/etc/kube-flannel"
    - name: flannel-docker-opts.service
      enabled: true
      contents: |
        [Unit]
        PartOf=flanneld.service
        Requires=flanneld.service
        After=flanneld.service
        [Service]
        Type=oneshot
        ExecStart=/bin/true
    - name: kubelet.service
      enabled: true
      contents: |
        [Unit]
        Description=Kubelet via Hyperkube ACI
        After=network-online.target nss-lookup.target
        Wants=network-online.target nss-lookup.target
        [Service]
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/run/kubelet-pod.uuid \
          --inherit-env \
          --dns=host \
          --net=host \
          --volume var-lib-cni,kind=host,source=/var/lib/cni \
          --volume var-log,kind=host,source=/var/log \
          --volume etc-machine-id,kind=host,source=/etc/machine-id,readOnly=true \
          --volume modprobe,kind=host,source=/usr/sbin/modprobe \
          --mount volume=var-lib-cni,target=/var/lib/cni \
          --mount volume=var-log,target=/var/log \
          --mount volume=etc-machine-id,target=/etc/machine-id \
          --mount volume=modprobe,target=/usr/sbin/modprobe \
{{- if .CalicoNetworking }}
          --volume var-lib-calico,kind=host,source=/var/lib/calico,readOnly=true \
          --volume etc-cni,kind=host,source=/etc/cni,readOnly=true \
          --volume opt-cni,kind=host,source=/opt/cni,readOnly=true \
          --mount volume=var-lib-calico,target=/var/lib/calico \
          --mount volume=etc-cni,target=/etc/cni \
          --mount volume=opt-cni,target=/opt/cni \
{{- end }}
          --insecure-options=image"
        Environment="KUBELET_IMAGE_TAG={{ .HyperkubeImageTag }}"
        Environment="KUBELET_IMAGE_URL=docker://{{ .HyperkubeImage }}"
        Environment="KUBELET_IMAGE_ARGS=--name=kubelet --exec=/kubelet"
        ExecStartPre=/usr/bin/host identity-3.{{ .OpenstackRegion }}.cloud.sap
{{- if .CalicoNetworking }}
        ExecStartPre=/bin/mkdir -p /etc/cni /opt/cni /var/lib/calico
 {{- end }}
        ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
        ExecStartPre=/bin/mkdir -p /var/lib/cni
        ExecStartPre=-/opt/bin/rkt rm --uuid-file=/var/run/kubelet-pod.uuid
        ExecStart=/opt/bin/kubelet-wrapper \
          --cert-dir=/var/lib/kubelet/pki \
          --cloud-config=/etc/kubernetes/openstack/openstack.config \
          --cloud-provider=openstack \
          --config=/etc/kubernetes/kubelet/config \
          --bootstrap-kubeconfig=/etc/kubernetes/bootstrap/kubeconfig \
          --hostname-override={{ .NodeName }} \
          --kubeconfig=/var/lib/kubelet/kubeconfig \
{{- if .CalicoNetworking }}
          --network-plugin=cni \
{{- else }}
          --network-plugin=kubenet \
          --network-plugin-mtu=8900 \
{{- end }}
          --non-masquerade-cidr=0.0.0.0/0 \
          --lock-file=/var/run/lock/kubelet.lock \
          --pod-infra-container-image={{ .PauseImage }}:{{ .PauseImageTag }} \
{{- if .NodeLabels }}
          --node-labels={{ .NodeLabels | join "," }} \
{{- end }}
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
      enabled: true
      contents: |
        [Path]
        PathExists=/var/lib/kubelet/kubeconfig
        [Install]
        WantedBy=multi-user.target
    - name: kube-proxy.service
      enabled: true
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
          docker://{{ .HyperkubeImage }}:{{ .HyperkubeImageTag }} \
          --name kube-proxy \
          --exec=/hyperkube \
          -- \
          kube-proxy \
          --config=/etc/kubernetes/kube-proxy/config
        ExecStopPost=/opt/bin/rkt gc --mark-only
        KillMode=mixed
        Restart=always
        RestartSec=10s
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
    - name: rkt-gc.service
      contents: |
        [Unit]
        Description=Garbage Collection for rkt

        [Service]
        Environment=GRACE_PERIOD=24h
        Type=oneshot
        ExecStart=/opt/bin/rkt gc --grace-period=${GRACE_PERIOD}
    - name: rkt-gc.timer
      enabled: true
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
    - path: /var/lib/iptables/rules-save
      filesystem: root
      mode: 0644
      overwrite: true
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
    - path: /etc/kube-flannel/net-conf.json
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          {
            "Network": "{{ .ClusterCIDR }}",
            "Backend": {
               "Type": "host-gw"
            }
          }
    - path: /etc/kubernetes/environment
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
          NODE_NAME={{ .NodeName }}
    - path: /etc/kubernetes/certs/kubelet-clients-ca.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
{{ .KubeletClientsCA | indent 10 }}
    - path: /etc/kubernetes/certs/apiserver-clients-system-kube-proxy-key.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
{{ .ApiserverClientsSystemKubeProxyKey | indent 10 }}
    - path: /etc/kubernetes/certs/apiserver-clients-system-kube-proxy.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
{{ .ApiserverClientsSystemKubeProxy | indent 10 }}
    - path: /etc/kubernetes/certs/tls-ca.pem
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        inline: |-
{{ .TLSCA | indent 10 }}
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
      overwrite: true
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
          featureGates:
            NodeLease: false
    - path: /etc/kubernetes/kube-proxy/config
      filesystem: root
      mode: 0644
      overwrite: true
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
          featureGates: {}
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
      overwrite: true
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
      overwrite: true
      contents:
        inline: |-
          REBOOT_STRATEGY="off"
    - path: /opt/bin/rkt
      filesystem: root
      mode: 0755
      overwrite: true
      contents:
        source: https://repo.{{.OpenstackRegion}}.cloud.sap/controlplane/flatcar-rkt/rkt-v1.30.0.gz
        compression: gzip
        verification:
          hash: sha512-259fd4d1e1d33715c03ec1168af42962962cf70abc5ae9976cf439949f3bcdaf97110455fcf40c415a2adece28f6a52b46f8abd180cad1ee2e802d41a590b35f
    - path: /opt/rkt/stage1-fly.aci
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        source: https://repo.{{.OpenstackRegion}}.cloud.sap/controlplane/flatcar-rkt/stage1-fly-rkt-v1.30.0.aci
        verification:
          hash: sha512-624bcf48b6829d2ac05c5744996d0fbbe2a0757bf2e5ad859f962a7001bb81980b0aa7be8532f3ec1ef7bbf025bbd089f5aa2eee9fdadefed1602343624750f1
    - path: /opt/rkt/stage1-coreos.aci
      filesystem: root
      mode: 0644
      overwrite: true
      contents:
        source: https://repo.{{.OpenstackRegion}}.cloud.sap/controlplane/flatcar-rkt/stage1-coreos-rkt-v1.30.0.aci
        verification:
          hash: sha512-b295e35daab8ca312aeb516a59e79781fd8661d585ecd6c2714bbdec9738ee9012114a2ec886b19cb6eb2e212d72da6f902f02ca889394ef23dbd81fbf147f8c
    - path: /etc/rkt/paths.d/stage1.json
      filesystem: root
      mode: 0644
      overwrite: true
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
      overwrite: true
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
          mkdir --parents /var/lib/docker
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
            --volume coreos-var-lib-docker,kind=host,source=/var/lib/docker,readOnly=false \
            --volume coreos-var-lib-kubelet,kind=host,source=/var/lib/kubelet,readOnly=false,recursive=true \
            --volume coreos-var-log,kind=host,source=/var/log,readOnly=false \
            --volume coreos-os-release,kind=host,source=/usr/lib/os-release,readOnly=true \
            --volume coreos-run,kind=host,source=/run,readOnly=false \
            --volume coreos-lib-modules,kind=host,source=/lib/modules,readOnly=true \
            --volume coreos-etc-machine-id,kind=host,source=/etc/machine-id,readOnly=true \
            --mount volume=coreos-etc-kubernetes,target=/etc/kubernetes \
            --mount volume=coreos-etc-ssl-certs,target=/etc/ssl/certs \
            --mount volume=coreos-usr-share-certs,target=/usr/share/ca-certificates \
            --mount volume=coreos-var-lib-docker,target=/var/lib/docker \
            --mount volume=coreos-var-lib-kubelet,target=/var/lib/kubelet \
            --mount volume=coreos-var-log,target=/var/log \
            --mount volume=coreos-os-release,target=/etc/os-release \
            --mount volume=coreos-run,target=/run \
            --mount volume=coreos-lib-modules,target=/lib/modules \
            --mount volume=coreos-etc-machine-id,target=/etc/machine-id \
            --hosts-entry host \
            ${RKT_STAGE1_ARG} \
            ${KUBELET_IMAGE} \
              ${KUBELET_IMAGE_ARGS} \
              -- "$@"

    - path: /opt/bin/flannel-wrapper
      filesystem: root
      mode: 0755
      overwrite: true
      contents:
        inline: |-
          #!/bin/bash -e
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
          if [[ -n "${FLANNEL_VER}" ]]; then
            echo FLANNEL_VER environment variable is deprecated, please use FLANNEL_IMAGE_TAG instead
          fi
          if [[ -n "${FLANNEL_IMG}" ]]; then
            echo FLANNEL_IMG environment variable is deprecated, please use FLANNEL_IMAGE_URL instead
          fi
          FLANNEL_IMAGE_TAG="${FLANNEL_IMAGE_TAG:-${FLANNEL_VER}}"
          require_ev_one FLANNEL_IMAGE FLANNEL_IMAGE_TAG
          FLANNEL_IMAGE_URL="${FLANNEL_IMAGE_URL:-${FLANNEL_IMG:-docker://quay.io/coreos/flannel}}"
          FLANNEL_IMAGE="${FLANNEL_IMAGE:-${FLANNEL_IMAGE_URL}:${FLANNEL_IMAGE_TAG}}"
          if [[ "${FLANNEL_IMAGE%%/*}" == "quay.io" ]] && ! (echo "${RKT_RUN_ARGS}" | grep -q trust-keys-from-https); then
            RKT_RUN_ARGS="${RKT_RUN_ARGS} --trust-keys-from-https"
          elif [[ "${FLANNEL_IMAGE%%/*}" == "docker:" ]] && ! (echo "${RKT_RUN_ARGS}" | grep -q insecure-options); then
            RKT_RUN_ARGS="${RKT_RUN_ARGS} --insecure-options=image"
          fi
          ETCD_SSL_DIR="${ETCD_SSL_DIR:-/etc/ssl/etcd}"
          if [[ -d "${ETCD_SSL_DIR}" ]]; then
            RKT_RUN_ARGS="${RKT_RUN_ARGS} \
              --volume coreos-ssl,kind=host,source=${ETCD_SSL_DIR},readOnly=true \
              --mount volume=coreos-ssl,target=${ETCD_SSL_DIR} \
            "
          fi
          if [[ -S "${NOTIFY_SOCKET}" ]]; then
            RKT_RUN_ARGS="${RKT_RUN_ARGS} \
              --mount volume=coreos-notify,target=/run/systemd/notify \
              --volume coreos-notify,kind=host,source=${NOTIFY_SOCKET} \
              --set-env=NOTIFY_SOCKET=/run/systemd/notify \
            "
          fi
          mkdir --parents /run/flannel
          RKT="${RKT:-/opt/bin/rkt}"
          RKT_STAGE1_ARG="${RKT_STAGE1_ARG:---stage1-from-dir=stage1-fly.aci}"
          set -x
          exec ${RKT} ${RKT_GLOBAL_ARGS} \
            run ${RKT_RUN_ARGS} \
            --net=host \
            --volume coreos-run-flannel,kind=host,source=/run/flannel,readOnly=false \
            --volume coreos-etc-ssl-certs,kind=host,source=/etc/ssl/certs,readOnly=true \
            --volume coreos-usr-share-certs,kind=host,source=/usr/share/ca-certificates,readOnly=true \
            --volume coreos-etc-hosts,kind=host,source=/etc/hosts,readOnly=true \
            --volume coreos-etc-resolv,kind=host,source=/etc/resolv.conf,readOnly=true \
            --mount volume=coreos-run-flannel,target=/run/flannel \
            --mount volume=coreos-etc-ssl-certs,target=/etc/ssl/certs \
            --mount volume=coreos-usr-share-certs,target=/usr/share/ca-certificates \
            --mount volume=coreos-etc-hosts,target=/etc/hosts  \
            --mount volume=coreos-etc-resolv,target=/etc/resolv.conf \
            --inherit-env \
            ${RKT_STAGE1_ARG} \
            ${FLANNEL_IMAGE} \
              ${FLANNEL_IMAGE_ARGS} \
              -- "$@"
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
`
