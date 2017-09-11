package templates

var Node = `
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
    - name: kubelet.service
      enable: true
      contents: |
        [Unit]
        Description=Kubelet via Hyperkube ACI

        [Service]
        Environment="RKT_RUN_ARGS=--uuid-file-save=/var/run/kubelet-pod.uuid \
          --volume=resolv,kind=host,source=/etc/resolv.conf \
          --mount volume=resolv,target=/etc/resolv.conf \
          --volume var-log,kind=host,source=/var/log \
          --mount volume=var-log,target=/var/log"
        Environment="KUBELET_IMAGE_TAG=v1.7.5_coreos.0"
        Environment="KUBELET_IMAGE_URL=quay.io/coreos/hyperkube"
        ExecStartPre=/bin/mkdir -p /etc/kubernetes/manifests
        ExecStartPre=/bin/mkdir -p /srv/kubernetes/manifests
        ExecStartPre=-/usr/bin/rkt rm --uuid-file=/var/run/kubelet-pod.uuid
        ExecStart=/usr/lib/coreos/kubelet-wrapper \
          --cloud-config=/etc/kubernetes/openstack/openstack.config \
          --cloud-provider=openstack \
          --require-kubeconfig \
          --bootstrap-kubeconfig=/etc/kubernetes/bootstrap/kubeconfig \
          --network-plugin=kubenet \
          --lock-file=/var/run/lock/kubelet.lock \
          --exit-on-lock-contention \
          --pod-manifest-path=/etc/kubernetes/manifests \
          --allow-privileged \
          --cluster_domain=cluster.local \
          --client-ca-file=/etc/kubernetes/certs/kube-clients/ca.pem \
          --anonymous-auth=false
        ExecStop=-/usr/bin/rkt stop --uuid-file=/var/run/kubelet-pod.uuid
        Restart=always
        RestartSec=10

        [Install]
        WantedBy=multi-user.target

storage:
  files:
    - path: /etc/kubernetes/certs/kube-clients/ca.pem
      filesystem: root
      mode: 0644
      contents: 
        inline: |-
{{ .ApiserverClientsCA | indent 10 }}
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
