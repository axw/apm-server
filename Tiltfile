# vim: set ft=starlark:
load('ext://helm_remote', 'helm_remote')
load('ext://helm_resource', 'helm_resource', 'helm_repo')
load('ext://namespace', 'namespace_create')

def printup(*args):
  if config.tilt_subcommand == "up":
    print(*args)

config.define_bool('local-kibana')
config.define_string('elasticsearch-repo')

parsed_config = config.parse()

script_dir = os.path.join(config.main_dir, 'script')
elasticsearch_repo_dir = parsed_config.get('elasticsearch-repo', None)

default_kibana_port = 5601
kibana_host = "localhost"
kibana_base_path = ""
golang_version = str(read_file('./.go-version')).strip()

# checks whether (dev mode) kibana is already running at given port,
# and returns the URL it is redirected to
def discover_kibana():

    default_kibana_host = "localhost"

    count = int(
      str(
        local("lsof -n -i :%s | grep LISTEN | wc -l | tr -d ' '" % default_kibana_port, quiet=True, echo_off=True)
      ).strip()
    )

    if count <= 0:
      fail("No service found on port " + str(default_kibana_port))
      return None


    kibana_redirect_url = str(
      local(
        "curl --fail -s -I -o /dev/null -w '%{redirect_url}' " + 'http://{}:{}'.format(default_kibana_host, default_kibana_port) + ' || [ $? -eq 7 ]',
        quiet=True,
        echo_off=True
      )
    ).strip().replace("http://", "")

    return kibana_redirect_url

custom_build(
  'apm-server',
  'DOCKER_BUILDKIT=1 docker build -t $EXPECTED_REF --build-arg GOLANG_VERSION=%s -f packaging/docker/Dockerfile .' % golang_version,
  deps = ['.'],
  ignore = ['**/*_test.go', 'tools/**', 'systemtest/**', 'docs/**'],
)
helm_set=['apm_server.image=apm-server']

# If elasticsearch-repo is configured, build a custom Elasticsearch
# image using the build(Arch)DockerImage target.
if elasticsearch_repo_dir != None:
  # Elasticsearch images are always tagged with "x86_64" or "aarch64"
  goarch = str(local('go env GOARCH', quiet=True)).strip()
  (task, tag) = {
    'amd64': ('buildDockerImage', 'x86_64'),
    'arm64': ('buildAarch64DockerImage', 'aarch64'),
  }[goarch]
  gradle_task = 'build{}DockerImage'.format(tag.title())
  custom_build(
    'elasticsearch',
    './gradlew ' + task,
    tag=tag,
    dir=elasticsearch_repo_dir,
    deps=[
      os.path.join(elasticsearch_repo_dir, 'server'),
      os.path.join(elasticsearch_repo_dir, 'x-pack')
    ],
  )
  helm_set.append('elasticsearch.image=elasticsearch')
  k8s_kind('Elasticsearch', image_json_path='{.spec.image}')
else:
  k8s_kind('Elasticsearch')

namespace_create('elastic-system')
helm_repo('elastic', 'https://helm.elastic.co/', labels='stack')
helm_resource('eck-operator', 'elastic/eck-operator', namespace='elastic-system', labels='stack')
k8s_yaml(helm('testing/infra/k8s/helm', set=helm_set))

k8s_kind('ApmServer', image_json_path='{.spec.image}')
k8s_resource('eck-operator', objects=['eck-trial-license:Secret:elastic-system'], labels='stack')
k8s_resource('apm-server', port_forwards=8200, labels='apm-server')
k8s_resource('elasticsearch', port_forwards=9200, objects=['elasticsearch-admin:Secret:default'], labels='stack')

# Conditionally set up Kibana resource
discovered_kibana_url = discover_kibana() if "local-kibana" in parsed_config else None

if discovered_kibana_url and config.tilt_subcommand == "up":
  printup("Waiting until local Kibana is ready at {}".format(discovered_kibana_url))
  
  # get only the path from the URL so we can pass it separately to APM Server
  url = discovered_kibana_url
  if "://" in discovered_kibana_url:
    url = url.split("://")[1]
  kibana_base_path = url.split("/", 1)[1] if "/" in url else ""


  local_resource(
    "init_kibana_system_user",
    cmd = """curl -s -X POST -H "Content-Type: application/json" -d '{"username":"kibana_system_user", "password":"changeme", "roles": [ "kibana_system"]}' http://admin:changeme@localhost:9200/_security/user/kibana_system_user""",
    resource_deps=["elasticsearch"],
    labels='stack',
  )

  local_resource(
    'kibana',
    cmd=[ os.path.join(script_dir, 'wait_for_kibana.sh'), "http://admin:changeme@{}".format(discovered_kibana_url)],
    resource_deps=["init_kibana_system_user"],
    links=["http://localhost:5601"],
    labels='stack',
  )

else:
  printup("Starting Kibana as a container")
  k8s_kind('Kibana')
  k8s_resource('kibana', port_forwards=default_kibana_port, resource_deps=['elasticsearch'], labels='stack')

# Delete ECK entirely on `tilt down`, to ensure `tilt up` starts from a clean slate.
#
# Without this, ECK refuses to recreate the trial license. That's expected
# for actual trial use, but we rely an Enterprise feature for local development:
# installing the integration package by uploading to Fleet.
if config.tilt_subcommand == "down":
  print(local("kubectl delete --ignore-not-found namespace/elastic-system"))

# Add a button for sending trace events and metrics to APM Server.
run_with_go_ver = os.path.join(script_dir, 'run_with_go_ver')
load('ext://uibutton', 'cmd_button')
cmd_button(
  'apm-server:sendotlp',
  argv=['sh', '-c', 'cd systemtest && %s go run ./cmd/sendotlp' % run_with_go_ver],
  resource='apm-server',
  icon_name='input',
  text='Send OTLP data',
)
