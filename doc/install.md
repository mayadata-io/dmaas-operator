## Installing DMaaS Operator

To install dmaas-operator, run 

```bash
kubectl apply -f deploy/dmaas-operator.yaml
```

Above command will install the following components:
1. All CRDs for dmaas-operator
    - DMaaSRestore
    - DMaaSBackup
    - PreBackupAction

2. Namespace(=dmaas) for dmaas-operator.

3. ServiceAccount(=dmaas-operator) for dmaas-operator deployment.

4. ClusterRoleBinding for above ServiceAccount using ClusterRole=cluster-admin.

5. Deployment(=dmaas-operator) with replica=1.
    - Deployment uses following environment variable:
        - OPENEBS_NAMESPACE : value is set to openebs.
        - VELERO_NAMESPACE : value is set to velero.
        - NAMESPACE: value is set to deployment namespace(=metadata.namespace) .

