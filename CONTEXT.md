# Downlink

A Kubernetes-native synthetic monitoring platform. Teams write standard Playwright tests, deploy them as CRDs, and run them inside their own cluster on any schedule. Metrics flow into Prometheus; traces and screenshots go to S3. The operator owns the execution infrastructure; users own the browser compute.

## Language

### Core objects

**SyntheticTest**:
A Kubernetes Custom Resource that defines a synthetic monitoring test — its source, schedule, container config, and artifact storage settings. The unit of configuration for a test.
_Avoid_: test definition, monitor, check

**SyntheticTestRun**:
A Kubernetes Custom Resource that records the outcome of a single execution of a SyntheticTest. Created by the operator when an Argo Workflow with the `downlinklabs.io/synthetic-test` label completes. The unit of run history.
_Avoid_: test result, execution record, workflow run

**Probe**:
The runtime concept of a SyntheticTest executing against a target — what is being measured. Used in metric label context ("probe metrics", "probe frequency"). Not a Kubernetes object.
_Avoid_: test run (use SyntheticTestRun for the record), monitor check

**Runner**:
The pre-built container image that provides the Playwright runtime, browsers, and the Downlink reporter. Mounted with test source at startup. Users do not build or maintain this image.
_Avoid_: test container, playwright image

### Source models

**Inline Source**:
Test code embedded directly in `SyntheticTest.Spec.Source.Inline` as a string. Stored in etcd via a ConfigMap. Limited to a single file and ~1 MB. The only implemented source model; appropriate for onboarding and simple probes.
_Avoid_: embedded source, inline test

**S3 Source** _(planned)_:
A versioned test bundle stored in object storage, referenced by bucket and key in `SyntheticTest.Spec.Source.S3`. Downloaded by an init container at pod startup. Supports full multi-file suites and large test codebases.
_Avoid_: bundle source, remote source

### Execution

**CronWorkflow**:
The Argo Workflows object that implements the scheduled execution of a SyntheticTest. Owned and reconciled by the operator. One CronWorkflow per SyntheticTest.
_Avoid_: scheduled workflow, cron job

**Workflow**:
A one-shot Argo Workflows execution. Used for on-demand and CI-triggered runs. Distinct from CronWorkflow — no schedule, fires once.
_Avoid_: manual run, ad-hoc execution

**Reporter**:
The Playwright reporter plugin (`@downlink/synthetics/reporter`) baked into the Runner image. Subscribes to Playwright lifecycle hooks and pushes `synthetic_test_duration_seconds` and `synthetic_step_duration_seconds` histograms to the Pushgateway. Test authors write zero instrumentation code.
_Avoid_: metrics plugin, instrumentation layer

### Personas

**Cluster Operator**:
Installs and maintains the Downlink Helm chart and operator in a Kubernetes cluster. Owns infrastructure decisions: artifact storage, Pushgateway wiring, namespace configuration.
_Avoid_: platform engineer (too broad), admin

**Developer**:
Writes SyntheticTests to monitor critical user paths. Consumes probe metrics for SLO definitions and alerts. Does not manage cluster infrastructure.
_Avoid_: test author (too narrow — they also own the SLO)

**SRE**:
Writes SyntheticTests for systems that are not well-instrumented at the application layer. Cross-functional; may or may not own the cluster.
_Avoid_: platform SRE (too narrow)

## Example dialogue

> "The checkout probe failed twice in the last hour."
> "Was it the SyntheticTest failing or the CronWorkflow not scheduling?"
> "The SyntheticTestRun shows phase Failed — the Workflow ran, the test itself failed. Screenshots are in the ArtifactURL."
> "OK, so the Runner started fine. Can we trigger a one-shot Workflow to retest after the hotfix?"
