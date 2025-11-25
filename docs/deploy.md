# deploy

[Deploy external-dns using its Helm chart](https://github.com/kubernetes-sigs/external-dns/tree/cfe74817e31aff957511934f764ac14a937e8ece/charts/external-dns) with the following values:

```yaml
provider:
  name: webhook
  webhook:
    image:
      repository: ghcr.io/frantjc/external-dns-dnsserver-webhook
      tag: 0.1.4
    args:
      - --debug
      - --dns-port=5353
    securityContext:
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - ALL
      privileged: false
      runAsNonRoot: true
      runAsGroup: 1001
      runAsUser: 1001
```

Then, ensure that the resulting Deployment gets a Service created to expose it, e.g. with a LoadBalancer:

```yaml
---
apiVersion: v1
kind: Service
metadata:
  name: dnsserver
spec:
  type: LoadBalancer
  ports:
    - port: 53
      targetPort: 5353
      protocol: UDP
  selector:
    app.kubernetes.io/name: external-dns
```

Next, create an Ingress or a Service for external-dns to reconcile. Finally, ensure that the dnsserver Service has the expected record.
