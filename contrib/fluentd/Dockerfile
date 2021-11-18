FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/fluentd:v1.14-1

LABEL source_repository=https://github.com/fluent/fluentd
USER root
RUN fluent-gem install fluent-plugin-elasticsearch
USER fluent
