# kubernetes-prometheus-autoscaler

Scale deployments based on prometheus queries

### How to use

Run this as a pod in your cluster.  The [container image](https://hub.docker.com/r/joeelliott/kubernetes-prometheus-scaler) is available on DockerHub.

Add this label to deployments you want considered for scaling:

```
  labels:
    scale: prometheus
```

There are two different scaling strategies currently available.  See below for the annotations necessary to control scaling.

#### Step Scaling

With step scaling you provide a query and two conditions.  The two conditions `scale-up-when` and `scale-down-when` are evaluated to determine whether or not the deployment should be scaled up or down by 1 replica.

```
    prometheusScaler/prometheus-query: "time() % 60"
    prometheusScaler/min-scale: "2"
    prometheusScaler/max-scale: "5"
    prometheusScaler/scale-up-when: "result > 50"
    prometheusScaler/scale-down-when: "result < 10"
```

#### Direct Scaling

With direct scaling you provide a query and one condition.  The replica count is set to the value retrieved from evaluating the `scale-to` expression directly.

```
    prometheusScaler/prometheus-query: "time() % 60"
    prometheusScaler/min-scale: "2"
    prometheusScaler/max-scale: "5"
    prometheusScaler/scale-to: "result"
```

#### Relative Scaling

With relative scaling you provide a query and one condition.  The replica count is set to the value retrieved from evaluating the `scale-relative` expression and adding it to the current number of replicas.

```
    prometheusScaler/prometheus-query: "time() % 3 - 1"
    prometheusScaler/min-scale: "2"
    prometheusScaler/max-scale: "5"
    prometheusScaler/scale-relative: "result"
```

Scale up, scale down, scale relative and scale to conditions use this clever repo https://github.com/Knetic/govaluate.  The value retrieved from the query is exposed to the expression as a parameter named `result`.

#### Command Line Usage

```
  -assessment-interval duration
        Time to sleep between checking deployments. (default 1m0s)
  -prometheus-url string
        URL to query. (default "http://prometheus:9090")
```

#### Prometheus Metrics

Publishes `prometheusscaler_error_total` on port 8080 at `/metrics`.  You can use this to alert on exceptions thrown while executing queries or attempting to scale.

### Improvements

This repo is still under active development and needs a long list of improvements.  Some obvious ones:

- Better logging
- Obvious Performance Improvements (Don't run scale up query if at max)
- Comments/documentation
- Publish to docker hub once it sucks less
- Refactor to use go funcs and channels?
- Add more scaling strategies?
- Add more testing
