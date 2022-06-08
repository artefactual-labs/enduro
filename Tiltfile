version_settings(constraint=">=0.22.2")

load("ext://uibutton", "cmd_button", "text_input")

docker_build("enduro:dev", context=".")

docker_build(
  "enduro-a3m-worker:dev",
  context=".",
  target="enduro-a3m-worker"
)

docker_build(
  "enduro-dashboard:dev",
  context="dashboard",
  target="builder",
  live_update=[
    fall_back_on("dashboard/vite.config.js"),
    sync("dashboard/", "/app/"),
    run(
      "npm set cache /app/.npm && npm install-clean",
      trigger=["dashboard/package.json", "dashboard/package-lock.json"]
    ),
  ]
)

k8s_yaml([
  "hack/kube/dev/enduro-a3m.yaml",
  "hack/kube/dev/enduro-dashboard.yaml",
  "hack/kube/dev/enduro.yaml",
  "hack/kube/dev/minio-setup-buckets-job.yaml",
  "hack/kube/dev/minio.yaml",
  "hack/kube/dev/mysql.yaml",
  "hack/kube/dev/opensearch-dashboards.yaml",
  "hack/kube/dev/opensearch.yaml",
  "hack/kube/dev/redis.yaml",
  "hack/kube/dev/temporal-ui.yaml",
  "hack/kube/dev/temporal.yaml",
])

k8s_resource("enduro-dashboard", port_forwards="3000")

k8s_resource("minio", port_forwards=["7460:9001", "0.0.0.0:7461:9000"])

k8s_resource("opensearch-dashboards", port_forwards="7500:5601")

k8s_resource("temporal-ui", port_forwards="7440:8080")

cmd_button(
  "minio-upload",
  argv=[
    "sh",
    "-c",
    "docker run --rm \
      --add-host=host-gateway:host-gateway \
      --entrypoint=/bin/bash \
      -v $HOST_PATH:/sampledata/$OBJECT_NAME \
      minio/mc -c ' \
        mc alias set enduro http://host-gateway:7461 minio minio123; \
        mc cp -r /sampledata/$OBJECT_NAME enduro/sips/$OBJECT_NAME; \
      ' \
    ",
  ],
  location="nav",
  icon_name="cloud_upload",
  text="Minio upload",
  inputs=[
    text_input("HOST_PATH", label="Host path"),
    text_input("OBJECT_NAME", label="Object name"),
  ]
)
