# OCI Mirror

Container and OCI Image Mirror.

`oci-mirror` lets you mirror container images or any other oci artefact between registries.
It is designed to run on a regular basis, as a [CronJob](https://kubernetes.io/docs/concepts/workloads/controllers/cron-jobs) in kubernetes for example.
Under the hood it uses [go-containerregistry](github.com/google/go-containerregistry) to copy images directly from one registry to another.

## Configuration

Configuration is done with a `yaml` configuration, defaults to `oci-mirror.yaml`.

## Quickstart

First create a `oci-mirror.yaml` which matches your needs, then run it with the following command:

```bash
docker run -it -v $PWD/oci-mirror.yaml:/oci-mirror.yaml --rm ghcr.io/metal-stack/oci-mirror mirror
```

## Kubernetes

There is a sample deployment manifest available, you can simple run:

```bash
kubectl apply -f deploy
```

## TODO

- [x] support purging
- [ ] eventually support http(s) artifacts to be stored as OCIs
- [ ] support Regex Match for image tags
- [ ] store a OCI artifact which reflects all stored images
