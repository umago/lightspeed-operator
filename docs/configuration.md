# Configuration

Everything is configured through the `OpenStackLightspeed` custom
resource (`lightspeed.openstack.org/v1beta1`). This page documents every
field in its `spec`.

## Core fields

| Field | Required | Description |
|-------|----------|-------------|
| `lightspeed.defaultModel` | Yes | Default model alias selected for inference. Must match one of `models[].name`. |
| `models` | Yes | List of configured models. Must contain at least one entry. |
| `models[].name` | Yes | Kubernetes-style model alias (used by `lightspeed.defaultModel`). |
| `models[].llmEndpoint` | Yes | URL of the LLM endpoint (e.g. `https://api.openai.com/v1`). Must start with `http://` or `https://`. |
| `models[].llmEndpointType` | Yes | Provider type. See {ref}`supported-providers`. |
| `models[].llmCredentials` | Yes | `Secret` name (same namespace) with the API token under key `apitoken`. |
| `models[].modelName` | Yes | Provider-native model name to use at the configured endpoint. |
| `models[].maxTokensForResponse` | No | Max response tokens for this model. Minimum `1`. Defaults to `2048`. |
| `models[].llmProjectID` | No | Required by some providers (e.g. WatsonX). |
| `models[].llmDeploymentName` | No | Required by some providers (e.g. Azure OpenAI). |
| `models[].llmAPIVersion` | No | Required by some providers (e.g. Azure OpenAI). |
| `tlsCACertBundle` | No | `ConfigMap` name (same namespace) with a CA bundle used by all model endpoints. |

(supported-providers)=
## Supported LLM providers (`models[].llmEndpointType`)

- `openai` — OpenAI-compatible endpoints (Ollama, vLLM, etc.)
- `azure_openai` — Azure OpenAI (needs `llmDeploymentName`, `llmAPIVersion`)
- `watsonx` — IBM watsonx.ai (needs `llmProjectID`)
- `rhoai_vllm` — vLLM via Red Hat OpenShift AI
- `rhelai_vllm` — vLLM via RHEL AI
- `gemini` — Google Gemini

> [!TIP]
> This list grows over time. Check
> `oc explain openstacklightspeed.spec.models.llmEndpointType` on your cluster
> for the current, authoritative list.

## Logging (`logging`)

| Field | Default | Description |
|-------|---------|-------------|
| `logging.ogxLogLevel` | `all=info` | OGX container. Standard level, or `component=level` pairs (e.g. `core=debug,providers=info`). |
| `logging.lightspeedStackLogLevel` | `INFO` | lightspeed-service-api container. `DEBUG`/`INFO`/`WARNING`/`ERROR`/`CRITICAL`. |
| `logging.dataverseExporterLogLevel` | `INFO` | Feedback/transcript exporter sidecar. Same values as above. |
| `logging.postgresLogLevel` | `INFO` | PostgreSQL container. `DEBUG` also logs every SQL statement. |

## Persistent storage (`database`)

PostgreSQL always gets a PersistentVolumeClaim — this field only overrides
its size/class, it doesn't control whether one exists:

```yaml
spec:
  database:
    size: "5Gi"                # default: 1Gi
    class: "my-storage-class"  # default: cluster's default StorageClass
```

## Container resources (`resources`)

Every container has a default request/limit. Setting one replaces its
default entirely:

```yaml
spec:
  resources:
    llamaStack:
      requests: {cpu: "500m", memory: "2Gi"}
      limits: {cpu: "2", memory: "8Gi"}
    lightspeedService:
      requests: {cpu: "250m", memory: "512Mi"}
      limits: {cpu: "1", memory: "2Gi"}
    postgres:
      requests: {cpu: "30m", memory: "300Mi"}
      limits: {cpu: "500m", memory: "2Gi"}
    okp:
      requests: {cpu: "500m", memory: "2Gi"}
      limits: {cpu: "2", memory: "4Gi"}
    consolePlugin:
      requests: {cpu: "50m", memory: "64Mi"}
      limits: {cpu: "200m", memory: "256Mi"}
    mcp:
      requests: {cpu: "50m", memory: "300Mi"}
      limits: {memory: "500Mi"}
```

(offline-knowledge-portal)=
## Offline Knowledge Portal (`okp`)

> [!IMPORTANT]
> OKP is deployed on **every** install — `spec.okp` configures it, it
> doesn't gate whether it's deployed. Pulling its image needs the same
> free `registry.redhat.io` account as {ref}`redhat-registry-access`.

```yaml
spec:
  okp: {}   # no access key: browse individual pages, full-text search doesn't work
```

```yaml
spec:
  okp:
    accessKey: okp-access-key-secret   # Secret key: "access_key"
```

- **No `accessKey`** (default) — you can navigate directly to and read
  individual documentation and product lifecycle pages. The full-text
  search index, Solutions, and Articles are encrypted and require a key,
  so keyword search across the corpus doesn't work. What upstream users
  run on.
- **With `accessKey`** — unlocks that search index plus the encrypted
  knowledgebase. Needs an active Red Hat Satellite subscription ([get one](https://access.redhat.com/offline/access)) — a bonus if you already
  have one, not something every user needs.

By default, **RAG grounding is OKP-only** — the bundled community
documentation is disabled unless you set `dev.okpRagOnly: false` (below).

(quota-enforcement)=
## Quota enforcement (`quotas`)

Configure one or more limiters to enable token quota enforcement. The
operator uses its managed PostgreSQL instance for quota storage. Omitting
`quotas` or leaving `limiters` empty disables enforcement.

```yaml
spec:
  quotas:
    limiters:
      - name: per-user-hourly
        type: userLimiter
        initialQuota: 1000
        quotaIncrease: 1000
        period: "1 hour"
      - name: cluster-daily
        type: clusterLimiter
        initialQuota: 100000
        quotaIncrease: 100000
        period: "1 day"
    scheduler:
      period: 10
    enableTokenHistory: true
```

Each entry in `limiters` requires these fields:

| Field | Description |
|-------|-------------|
| `name` | A human-readable limiter name. |
| `type` | `userLimiter` for a per-user quota, or `clusterLimiter` for one quota shared by the cluster. |
| `initialQuota` | Number of tokens granted when the limiter resets. Must be zero or greater. |
| `quotaIncrease` | Number of tokens added by the scheduler at each quota interval. Must be zero or greater. |
| `period` | Interval that controls when the limiter resets or increases, such as `"30 seconds"`, `"1 hour"`, `"1 day"`, or `"1 hour 30 minutes"`. |

`scheduler` is optional and configures the background process that checks
limiters for reset or increase and reconnects to the database after a
connection failure:

- `period`: check interval in seconds. Default: `5`.
- `databaseReconnectionCount`: number of database reconnection attempts.
  Default: `10`.
- `databaseReconnectionDelay`: delay in seconds between reconnection
  attempts. Default: `1`.

Set `enableTokenHistory: true` to record per-user, model, and provider token
usage for auditing. It does not affect enforcement and defaults to `false`.

## Developer / experimental options (`dev`)

> [!WARNING]
> Not part of the stable API — may change without notice.

```yaml
spec:
  dev:
    featureFlags:
      - rhoso_mcps   # enables the read-only MCP introspection sidecar
    okpChunkFilterQuery: "product:(*openstack* OR *openshift*)"  # example override
    okpRagOnly: false  # include bundled community docs too, not just OKP
    rhosMCPConfig: |
      debug: true
      workers: 4
```

- `okpChunkFilterQuery` and `okpRagOnly` take effect immediately, with
  no `featureFlags` entry needed — they're independent of
  `rhoso_mcps`. If unset, `okpChunkFilterQuery` auto-detects your
  OpenShift/RHOSO versions instead of using the literal example above.
- `rhoso_mcps` — the one flag that does need to be set. Deploys the MCP
  introspection sidecar, which is read-only **by default**. See
  {doc}`usage`.
- `rhosMCPConfig` is deep-merged on top of the operator's own defaults
  — it can override anything the default config sets, including the
  `allow_write` flags that keep introspection read-only. Only set this
  if you understand exactly what you're overriding.
