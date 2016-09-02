# etcdstats

Connect to a running etcd cluster and display the nodes that consume the largest amount of space by
value.

# Prerequisites

```
$ go get -u github.com/coreos/etcd/client
```

# Usage

```
Usage of etcdstats:
  -alsologtostderr
        log to standard error as well as files
  -cacert string
        CA certificate file (optional)
  -cert string
        client certificate file (optional)
  -key string
        client certificate key file (optional)
  -log_backtrace_at value
        when logging hits line file:N, emit a stack trace
  -log_dir string
        If non-empty, write log files in this directory
  -logtostderr
        log to standard error instead of files
  -n int
        display top n highest nodes (default 20)
  -prefix string
        directory prefix to summarize (default "/")
  -server string
        server url, e.g. https://127.0.0.1:2379 (required)
  -stderrthreshold value
        logs at or above this threshold go to stderr
  -summarize value
        summarize descendent nodes for the directory prefixed by this value instead of displaying these nodes; may specify multiple times for multiple directories
  -v value
        log level for V logs
  -vmodule value
        comma-separated list of pattern=N settings for file-filtered logging
```

# Example output

```
$ etcdstats -server https://127.0.0.1:4001 \
-cacert /var/lib/origin/openshift.local.config/master/ca.crt \
-cert /var/lib/origin/openshift.local.config/master/master.etcd-client.crt \
-key /var/lib/origin/openshift.local.config/master/master.etcd-client.key \
-n 20 \
-summarize /openshift.io/images \
-summarize /kubernetes.io/secrets

Top 20 highest etcd nodes by value size (excluding summarized items):
NODE                                                         CHILDREN  SIZE
/openshift.io/images                                         26        1992791
/kubernetes.io/secrets/openshift-infra                       60        395833
/kubernetes.io/secrets/default                               16        103823
/openshift.io/templates/openshift                            12        78802
/kubernetes.io/secrets/kube-system                           9         57305
/kubernetes.io/secrets/openshift                             9         57080
/kubernetes.io/secrets/myproject                             9         56857
/openshift.io/authorization/cluster/policies/default         N/A       47539
/openshift.io/authorization/cluster/policies                 1         47539
/openshift.io/imagestreams/openshift                         11        25326
/kubernetes.io/controllers/default                           2         19905
/kubernetes.io/events/default                                30        18622
/openshift.io/authorization/cluster/policybindings/:default  N/A       16292
/openshift.io/authorization/cluster/policybindings           1         16292
/kubernetes.io/controllers/default/router-1                  N/A       15017
/kubernetes.io/pods/default                                  2         11651
/openshift.io/templates/openshift/jenkins-pipeline-example   N/A       10934
/openshift.io/deploymentconfigs/default                      2         9219
/kubernetes.io/serviceaccounts/openshift-infra               20        9145
/openshift.io/templates/openshift/jenkins-persistent         N/A       8636

Total value size: 2943727
Value size excluding summarized items 280038
```

Note, in the example, I asked it to summarize /kubernetes.io/secrets. The way summarization works,
it excludes showing regular (non-directory) nodes under the prefix. It does not exclude showing
directories under the prefix. This is why you see /kubernetes.io/secrets/openshift-infra - it is a
directory. What you do not see are etcd nodes for the individual secrets.

# License
etcdstats is licensed under the [Apache License, Version 2.0](http://www.apache.org/licenses/).
