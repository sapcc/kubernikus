/* vim: set filetype=yaml : */

package templates

var Node_1_7 = `
passwd:
  users:
    - name:          core
      password_hash: {{ .LoginPassword }}
{{- if .LoginPublicKey }}
      ssh_authorized_keys:
        - {{ .LoginPublicKey | quote }}
{{- end }}
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
        ExecStart=/usr/bin/coreos-metadata --provider=openstack-metadata --attributes=/run/metadata/coreos --ssh-keys=core --hostname=/etc/hostname
        Restart=on-failure
        RestartSec=30
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
          --volume etc-machine-id,kind=host,source=/etc/machine-id,readOnly=true \
          --mount volume=var-lib-cni,target=/var/lib/cni \
          --mount volume=var-log,target=/var/log" \
          --mount volume=etc-machine-id,target=/etc/machine-id
        Environment="KUBELET_IMAGE_TAG=v1.7.5_coreos.0"
        Environment="KUBELET_IMAGE_URL=quay.io/coreos/hyperkube"
        Environment="KUBELET_IMAGE_ARGS=--name=kubelet --exec=/kubelet"
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
          --read-only-port=0 \
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
          --name wormhole --exec wormhole -- client --listen {{ .ApiserverIP }}:6443 --kubeconfig=/var/lib/kubelet/kubeconfig
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
          quay.io/coreos/hyperkube:v1.7.5_coreos.0 \
          --name=kube-proxy \
          --exec=/hyperkube \
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
        ConditionPathIsSymbolicLink=!/etc/ssl/certs/381107d7.0
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
          MIIGPTCCBCWgAwIBAgIKYQ4GNwAAAAAADDANBgkqhkiG9w0BAQsFADBOMQswCQYD
          VQQGEwJERTERMA8GA1UEBwwIV2FsbGRvcmYxDzANBgNVBAoMBlNBUCBBRzEbMBkG
          A1UEAwwSU0FQIEdsb2JhbCBSb290IENBMB4XDTE1MDMxNzA5MjQ1MVoXDTI1MDMx
          NzA5MzQ1MVowRDELMAkGA1UEBhMCREUxETAPBgNVBAcMCFdhbGxkb3JmMQwwCgYD
          VQQKDANTQVAxFDASBgNVBAMMC1NBUE5ldENBX0cyMIICIjANBgkqhkiG9w0BAQEF
          AAOCAg8AMIICCgKCAgEAjuP7Hj/1nVWfsCr8M/JX90s88IhdTLaoekrxpLNJ1W27
          ECUQogQF6HCu/RFD4uIoanH0oGItbmp2p8I0XVevHXnisxQGxBdkjz+a6ZyOcEVk
          cEGTcXev1i0R+MxM8Y2WW/LGDKKkYOoVRvA5ChhTLtX2UXnBLcRdf2lMMvEHd/nn
          KWEQ47ENC+uXd6UPxzE+JqVSVaVN+NNbXBJrI1ddNdEE3/++PSAmhF7BSeNWscs7
          w0MoPwHAGMvMHe9pas1xD3RsRFQkV01XiJqqUbf1OTdYAoUoXo9orPPrO7FMfXjZ
          RbzwzFtdKRlAFnKZOVf95MKlSo8WzhffKf7pQmuabGSLqSSXzIuCpxuPlNy7kwCX
          j5m8U1xGN7L2vlalKEG27rCLx/n6ctXAaKmQo3FM+cHim3ko/mOy+9GDwGIgToX3
          5SQPnmCSR19H3nYscT06ff5lgWfBzSQmBdv//rjYkk2ZeLnTMqDNXsgT7ac6LJlj
          WXAdfdK2+gvHruf7jskio29hYRb2//ti5jD3NM6LLyovo1GOVl0uJ0NYLsmjDUAJ
          dqqNzBocy/eV3L2Ky1L6DvtcQ1otmyvroqsL5JxziP0/gRTj/t170GC/aTxjUnhs
          7vDebVOT5nffxFsZwmolzTIeOsvM4rAnMu5Gf4Mna/SsMi9w/oeXFFc/b1We1a0C
          AwEAAaOCASUwggEhMAsGA1UdDwQEAwIBBjAdBgNVHQ4EFgQUOCSvjXUS/Dg/N4MQ
          r5A8/BshWv8wHwYDVR0jBBgwFoAUg8dB/Q4mTynBuHmOhnrhv7XXagMwSwYDVR0f
          BEQwQjBAoD6gPIY6aHR0cDovL2NkcC5wa2kuY28uc2FwLmNvbS9jZHAvU0FQJTIw
          R2xvYmFsJTIwUm9vdCUyMENBLmNybDBWBggrBgEFBQcBAQRKMEgwRgYIKwYBBQUH
          MAKGOmh0dHA6Ly9haWEucGtpLmNvLnNhcC5jb20vYWlhL1NBUCUyMEdsb2JhbCUy
          MFJvb3QlMjBDQS5jcnQwGQYJKwYBBAGCNxQCBAweCgBTAHUAYgBDAEEwEgYDVR0T
          AQH/BAgwBgEB/wIBADANBgkqhkiG9w0BAQsFAAOCAgEAGdBNALO509FQxcPhMCwE
          /eymAe9f2u6hXq0hMlQAuuRbpnxr0+57lcw/1eVFsT4slceh7+CHGCTCVHK1ELAd
          XQeibeQovsVx80BkugEG9PstCJpHnOAoWGjlZS2uWz89Y4O9nla+L9SCuK7tWI5Y
          +QuVhyGCD6FDIUCMlVADOLQV8Ffcm458q5S6eGViVa8Y7PNpvMyFfuUTLcUIhrZv
          eh4yjPSpz5uvQs7p/BJLXilEf3VsyXX5Q4ssibTS2aH2z7uF8gghfMvbLi7sS7oj
          XBEylxyaegwOBLtlmcbII8PoUAEAGJzdZ4kFCYjqZBMgXK9754LMpvkXDTVzy4OP
          emK5Il+t+B0VOV73T4yLamXG73qqt8QZndJ3ii7NGutv4SWhVYQ4s7MfjRwbFYlB
          z/N5eH3veBx9lJbV6uXHuNX3liGS8pNVNKPycfwlaGEbD2qZE0aZRU8OetuH1kVp
          jGqvWloPjj45iCGSCbG7FcY1gPVTEAreLjyINVH0pPve1HXcrnCV4PALT6HvoZoF
          bCuBKVgkSSoGgmasxjjjVIfMiOhkevDya52E5m0WnM1LD3ZoZzavsDSYguBP6MOV
          ViWNsVHocptphbEgdwvt3B75CDN4kf6MNZg2/t8bRhEQyK1FRy8NMeBnbRFnnEPe
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
    - path: /etc/coreos/docker-1.12
      filesystem: root
      contents:
        inline: yes
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
          create-monitor = yes
          monitor-delay = 1m
          monitor-timeout = 30s
          monitor-max-retries = 3

          [BlockStorage]
          trust-device-path = no

          [Route]
          router-id = {{ .OpenstackRouterID }}
`
