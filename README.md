# OpenSergo CRD adapter for Sentinel Go (POC version)

**NOTE**: The Kubernetes operator logic was embedded in the POC data-source. This is only for demo. Please refer to https://github.com/opensergo/opensergo-specification/issues/17 for the canonical way to integrate with OpenSergo SDK.

Generate boilerplate code and OpenSergo CRD YAML:

```shell
make generate && make manifests
```

Install OpenSergo CRDs with:

```shell
kubectl apply -f config/crd/bases
```

* [Example](./examples)