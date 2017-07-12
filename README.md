# kubernetes-prometheus-autoscaler

Scale deployments based on prometheus queries

### How to use

Run this as a pod in your cluster.

Add this label to deployments you want considered:

```
  labels:
    scale: prometheus
```

There are two different scaling strategies currently available.

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

Scale up, scale down and scale to conditions use this clever repo https://github.com/Knetic/govaluate.  The value retrieved from query is exposed to the expression as a parameter named `result`.

#### Command Line Usage

```
  -assessment-interval duration
        Time to sleep between checking deployments. (default 1m0s)
  -prometheus-url string
        URL to query. (default "http://prometheus:9090")
```

### Improvements

This repo is still under active development and needs a long list of improvements (but it works!).  Some obvious ones:

- Better logging
- Obvious Performance Improvements (Don't run scale up query if at max)
- Comments/documentation
- Publish to docker hub once it sucks less
- Refactor to use go funcs and channels?
- Add more scaling strategies?
- Add testing
