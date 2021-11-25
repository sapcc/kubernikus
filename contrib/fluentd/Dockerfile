FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/library/fluentd:v1.14-1

LABEL source_repository=https://github.com/fluent/fluentd
USER root
RUN fluent-gem install fluent-plugin-elasticsearch fluent-plugin-openstack
# Fix OpenStack Auth in Fluentd plugin
RUN apk add patch
COPY auth_options.diff /
RUN patch /usr/lib/ruby/gems/2.7.0/gems/fluent-plugin-openstack-2.0.1/lib/fluent/plugin/out_swift.rb auth_options.diff && rm auth_options.diff
USER fluent
