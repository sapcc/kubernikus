FROM keppel.eu-de-1.cloud.sap/ccloud-dockerhub-mirror/fluent/fluentd:v1.14.6-1.1

LABEL source_repository=https://github.com/fluent/fluentd
USER root
RUN fluent-gem install fluent-plugin-elasticsearch fluent-plugin-openstack
# Fix OpenStack Auth in Fluentd plugin
RUN apk add --no-cache patch
COPY auth_options.diff /
RUN patch /usr/lib/ruby/gems/2.7.0/gems/fluent-plugin-openstack-2.0.1/lib/fluent/plugin/out_swift.rb auth_options.diff && rm auth_options.diff
USER fluent
