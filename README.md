# Atlassian Status Exporter for Prometheus

This exporter can be used to turn the Atlassian application /status endpoint into metrics for scraping. This could be used for alert triggering, or dashboard generation.

From Atlassian

>Many load balancers require a URL to constantly check the health of their backends in order to automatically remove them from the pool. It's important to use a stable and fast URL for this, but lightweight enough to not consume unnecessary resources.
>
>Reference: [https://confluence.atlassian.com/enterprise/confluence-data-center-technical-overview-612959401.html](https://confluence.atlassian.com/enterprise/confluence-data-center-technical-overview-612959401.html)

## Status Table

| HTTP Status Code | Response entity       | Description |
| ---------------- | ---------------       | ----------- |
| 200              | {"state":"RUNNING"}   | Running normally |
| 500              | {"state":"ERROR"}     | An error state |
| 503              | {"state":"STARTING"}  | Application is starting |
| 503              | {"state":"STOPPING"}  | Application is stopping |
| 200              | {"state":"FIRST_RUN"} | Application is running for the first time and has not yet been configured |
| 404              |                       | Application failed to start up in an unexpected way (the web application failed to deploy) |

## Docker Build Example

```none
docker build . -t atlassian_status_exporter
```

## Docker Run Example

List Help

```none
docker run -it --rm atlassian_status_exporter -help
```

Simple run

```none
docker run -it --rm -p 9997:9997 atlassian_status_exporter -app.url="<bitbucket|confluence|jira>.domain.com"
```

## Prometheus Job

```none
- job_name: "atlassian_status_exporter"
  static_configs:
  - targets:
    - 'host.domain.com:9997'
```

## References

Thank you everyone that writes code and docs!

* [https://golang.org/](https://golang.org/)
* [https://rsmitty.github.io/Prometheus-Exporters/](https://rsmitty.github.io/Prometheus-Exporters/)
* [https://prometheus.io/](https://prometheus.io/)
* [https://github.com/Sirupsen/logrus](https://github.com/Sirupsen/logrus)
* [https://www.atlassian.com/](https://www.atlassian.com/)
