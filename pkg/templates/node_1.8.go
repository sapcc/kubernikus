/* vim: set filetype=yaml : */

package templates

var Node_1_8 = `
passwd:
  users:
    - name:          core
      password_hash: xyTGJkB462ewk
      ssh_authorized_keys: 
        - "ssh-rsa AAAAB3NzaC1yc2EAAAABIwAAAQEAvFapuevZeHFpFn438XMjvEQYd0wt7+tzUdAkMiSd007Tx1h79Xm9ZziDDUe4W6meinVOq93MAS/ER27hoVWGo2H/vn/Cz5M8xr2j5rQODnrF3RmfrJTbZAWaDN0JTq2lFjmCHhZJNhr+VQP1uw4z2ofMBP6MLybnLmm9ukzxFYZqCCyfEEUTCMA9SWywtTpGQp8VLM4INCxzBSCuyt3SO6PBvJSo4HoKg/sLvmRwpCVZth48PI0EUbJ72wp88Cw3bv8CLce2TOkLMwkE6NRN55w2aOyqP1G3vixHa6YcVaLlkQhJoJsBwE3rX5603y2KjOhMomqHfXxXn/3GKTWlsQ== michael.j.schmidt@gmail.com"

locksmith:
  reboot_strategy: "reboot"

systemd:
  units:
    - name: iptables-restore.service
      enable: true
    - name: ccloud-metadata.service
      contents: |
        [Unit]
        Description=Converged Cloud Metadata Agent

        [Service]
        Type=oneshot
        ExecStart=/usr/bin/coreos-metadata --provider=openstack-metadata --attributes=/run/metadata/coreos --ssh-keys=core --hostname=/etc/hostname
    - name: ccloud-metadata-hostname.service
      enable: true
      contents: |
        [Unit]
        Description=Workaround for coreos-metadata hostname bug
        Requires=ccloud-metadata.service
        After=ccloud-metadata.service

        [Service]
        Type=oneshot
        EnvironmentFile=/run/metadata/coreos
        ExecStart=/usr/bin/hostnamectl set-hostname ${COREOS_OPENSTACK_HOSTNAME}
        
        [Install]
        WantedBy=multi-user.target
    - name: docker.service
      enable: true
      dropins:
        - name: 20-docker-opts.conf
          contents: |
            [Service]
            Environment="DOCKER_OPTS=--log-opt max-size=5m --log-opt max-file=5 --ip-masq=false --iptables=false --bridge=none"
    - name: kubelet.service
      enable: true
      contents: |
        [Unit]
        Description=Kubelet via Hyperkube ACI

        [Service]
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/run/kubelet-pod.uuid \
          --inherit-env \
          --dns=host \
          --net=host \
          --volume var-lib-cni,kind=host,source=/var/lib/cni \
          --volume var-log,kind=host,source=/var/log \
          --mount volume=var-lib-cni,target=/var/lib/cni \
          --mount volume=var-log,target=/var/log"
        Environment="KUBELET_IMAGE_TAG=v1.8.5_coreos.0"
        Environment="KUBELET_IMAGE_URL=quay.io/coreos/hyperkube"
        ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
        ExecStartPre=/bin/mkdir -p /var/lib/cni
        ExecStartPre=-/usr/bin/rkt rm --uuid-file=/var/run/kubelet-pod.uuid
        ExecStart=/usr/lib/coreos/kubelet-wrapper \
          --cert-dir=/var/lib/kubelet/pki \
          --cloud-config=/etc/kubernetes/openstack/openstack.config \
          --cloud-provider=openstack \
          --require-kubeconfig \
          --bootstrap-kubeconfig=/etc/kubernetes/bootstrap/kubeconfig \
          --network-plugin=kubenet \
          --lock-file=/var/run/lock/kubelet.lock \
          --exit-on-lock-contention \
          --pod-manifest-path=/etc/kubernetes/manifests \
          --allow-privileged \
          --cluster-dns={{ .ClusterDNSAddress }} \
          --cluster-domain={{ .ClusterDomain }} \
          --client-ca-file=/etc/kubernetes/certs/kubelet-clients-ca.pem \
          --non-masquerade-cidr=0.0.0.0/0 \
          --anonymous-auth=false
        ExecStop=-/usr/bin/rkt stop --uuid-file=/var/run/kubelet-pod.uuid
        Restart=always
        RestartSec=10

        [Install]
        WantedBy=multi-user.target
    - name: wormhole.service
      contents: |
        [Unit]
        Description=Kubernikus Wormhole
        Requires=network-online.target
        After=network-online.target
        [Service]
        Slice=machine.slice
        ExecStartPre=/usr/bin/rkt fetch --insecure-options=image --pull-policy=new docker://{{ .KubernikusImage }}:{{ .KubernikusImageTag }}
        ExecStart=/usr/bin/rkt run \
          --inherit-env \
          --net=host \
          --dns=host \
          --volume var-lib-kubelet,kind=host,source=/var/lib/kubelet,readOnly=true \
          --mount volume=var-lib-kubelet,target=/var/lib/kubelet \
          --volume etc-kubernetes-certs,kind=host,source=/etc/kubernetes/certs,readOnly=true \
          --mount volume=etc-kubernetes-certs,target=/etc/kubernetes/certs \
          docker://{{ .KubernikusImage }}:{{ .KubernikusImageTag }} \
          --exec wormhole -- client --listen {{ .ApiserverIP }}:6443 --kubeconfig=/var/lib/kubelet/kubeconfig
        ExecStopPost=/usr/bin/rkt gc --mark-only
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
        Requires=network-online.target
        After=network-online.target
        [Service]
        Slice=machine.slice
        ExecStart=/usr/bin/rkt run \
          --trust-keys-from-https \
          --inherit-env \
          --net=host \
          --dns=host \
          --volume etc-kubernetes,kind=host,source=/etc/kubernetes,readOnly=true \
          --mount volume=etc-kubernetes,target=/etc/kubernetes \
          --volume lib-modules,kind=host,source=/lib/modules,readOnly=true \
          --mount volume=lib-modules,target=/lib/modules \
          --stage1-from-dir=stage1-fly.aci \
          quay.io/coreos/hyperkube:v1.8.5_coreos.0 \
          --exec=hyperkube \
          -- \
          proxy \
          --config=/etc/kubernetes/kube-proxy/config
        ExecStopPost=/usr/bin/rkt gc --mark-only
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
        ConditionPathIsSymbolicLink=!/etc/ssl/certs/48b11003.0
        Before=early-docker.service docker.service
        [Service]
        ExecStart=/usr/sbin/update-ca-certificates
        RemainAfterExit=yes
        Type=oneshot
        [Install]
        WantedBy=multi-user.target

networkd:
  units:
    - name: 50-kubernikus.netdev
      contents: |
        [NetDev]
        Description=Kubernikus Dummy Interface
        Name=kubernikus
        Kind=dummy
    - name: 51-kubernikus.network
      contents: |
        [Match]
        Name=kubernikus
        [Network]
        DHCP=no
        Address={{ .ApiserverIP }}/32

storage:
  files:
    - path: /etc/ssl/certs/SAPNetCA_G2.pem
      filesystem: root
      mode: 0644
      contents:
        inline: |-
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
          -A POSTROUTING -p tcp ! -d {{ .ClusterCIDR }} -m addrtype ! --dst-type LOCAL -j MASQUERADE --to-ports 32000-65000
          -A POSTROUTING -p udp ! -d {{ .ClusterCIDR }} -m addrtype ! --dst-type LOCAL -j MASQUERADE --to-ports 32000-65000
          -A POSTROUTING -p icmp ! -d {{ .ClusterCIDR }} -m addrtype ! --dst-type LOCAL -j MASQUERADE
          COMMIT
    - path: /etc/sysctl.d/10-enable-icmp-redirects
      filesystem: root
      mode: 0644
      contents:
        inline: |-
          net.ipv4.conf.all.accept_redirects=1
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
    - path: /etc/kubernetes/kube-proxy/config
      filesystem: root
      mode: 0644
      contents: 
        inline: |-
          apiVersion: componentconfig/v1alpha1
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
          featureGates: ""
          healthzBindAddress: 0.0.0.0:10256
          hostnameOverride: ""
          iptables:
            masqueradeAll: false
            masqueradeBit: 14
            minSyncPeriod: 0s
            syncPeriod: 30s
          metricsBindAddress: 127.0.0.1:10249
          mode: ""
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
`
