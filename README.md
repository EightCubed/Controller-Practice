# Controller Practice

A controller which watches the CRD Logcleaner and performs logcleanup on specified persistent volumes claims based the spec.volumeNamePattern (regex)

The hack/controller-deployment.yaml file contains env that specifies the interval to clean up logs. In addition to this, the runLogCleanup function is executed when a logcleaner crd is added or updated.

### Sample logcleaners instance

```
apiVersion: apiextensions.k8s.io/v1
kind: CustomResourceDefinition
metadata:
  name: logcleaners.stable.example.com
spec:
  group: stable.example.com
  versions:
    - name: v1
      served: true
      storage: true
      schema:
        openAPIV3Schema:
          type: object
          properties:
            apiVersion:
              type: string
              enum: ["stable.example.com/v1"]
            kind:
              type: string
              enum: ["LogCleaner"]
            metadata:
              type: object
            spec:
              type: object
              properties:
                retentionPeriod:
                  type: integer
                targetNamespace:
                  type: string
                volumeNamePattern:
                  type: string
  scope: Namespaced
  names:
    plural: logcleaners
    singular: logcleaner
    kind: LogCleaner
    shortNames:
      - lc

```

## How it works?

The log cleanup function first fetches relevant Persisten Volume Claims and filter them using the regex. It then creates a pod attaching each of the PVCs and runs an exec function to delete logs older than the retention period ( spec.retentionPeriod )
