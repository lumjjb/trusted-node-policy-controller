## To run it on a standalone kubernetes cluster
1. Configure `kubectl` to point to a kubernetes cluster
2. Run following command to apply `trustednodepolicies.policies.ibm.com` CRD
```
kubectl apply -f deploy/crds/deploy/crds/policies.ibm.com_samplepolicies_crd.yaml
```
3. Run following command to update clusterrolebinding required by `Trusted Container Policy Controller`. Replace `<namespace>` in the command with the namespace where the controller is going to be deployed.
```
sed -i "" 's|namespace: default|namespace: <namespace>|g' deploy/cluster_role_binding.yaml
```
4. Run following command to deploy `Trusted Container Policy Controller`
```
kubectl apply -f deploy/
```
5. Run following command to create a sample trusted container policy
```
kubectl apply -f deploy/crds/policies.ibm.com_samplepolicies_cr.yaml
```
6. Label a node with `trusted=false` or use intel secl k8s controller to do so. This will trigger an event.

## To run it with IBM Multicloud Manager
1. Repeat step 1 to 4 on the managed cluster. Make sure you deploy them to cluster namespace. The namespace name is usually your cluster name
2. Run following command to create a MCM policy on hub cluster
```
kubectl apply -f deploy/crds/mcm-trustednodepolicy.yaml
```
3. Run step 6 on managed cluster to generate a violation
4. Then you should be able to see the policy and violation status on MCM console
