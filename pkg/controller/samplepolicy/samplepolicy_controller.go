// Copyright 2019 The Kubernetes Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package samplepolicy

import (
	"context"
	"fmt"

	//"reflect"
	"strings"
	"time"

	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	policiesv1alpha1 "github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1"
	"github.com/lumjjb/trusted-node-policy-controller/pkg/common"
	//testclient "k8s.io/client-go/kubernetes/fake"
)

var log = logf.Log.WithName("controller_samplepolicy")

// Finalizer used to ensure consistency when deleting a CRD
const Finalizer = "finalizer.policies.ibm.com"

const grcCategory = "system-and-information-integrity"

// availablePolicies is a cach all all available polices
var availablePolicies common.SyncedPolicyMap

// PlcChan a channel used to pass policies ready for update
var PlcChan chan *policiesv1alpha1.TrustedNodePolicy

// KubeClient a k8s client used for k8s native resources
var KubeClient *kubernetes.Interface

var reconcilingAgent *ReconcileTrustedNodePolicy

// NamespaceWatched defines which namespace we can watch for the GRC policies and ignore others
var NamespaceWatched string

// EventOnParent specifies if we also want to send events to the parent policy. Available options are yes/no/ifpresent
var EventOnParent string

// PrometheusAddr port addr for prom metrics
var PrometheusAddr string

// Add creates a new TrustedNodePolicy Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTrustedNodePolicy{client: mgr.GetClient(), scheme: mgr.GetScheme(), recorder: mgr.GetEventRecorderFor("samplepolicy-controller")}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("samplepolicy-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TrustedNodePolicy
	err = c.Watch(&source.Kind{Type: &policiesv1alpha1.TrustedNodePolicy{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	// Watch for changes to secondary resource Pods and requeue the owner TrustedNodePolicy
	err = c.Watch(&source.Kind{Type: &corev1.Node{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &policiesv1alpha1.TrustedNodePolicy{},
	})
	if err != nil {
		return err
	}

	return nil
}

// Initialize to initialize some controller variables
func Initialize(kClient *kubernetes.Interface, mgr manager.Manager, namespace, eventParent string) {
	KubeClient = kClient
	PlcChan = make(chan *policiesv1alpha1.TrustedNodePolicy, 100) //buffering up to 100 policies for update

	NamespaceWatched = namespace

	EventOnParent = strings.ToLower(eventParent)
}

// blank assignment to verify that ReconcileTrustedNodePolicy implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileTrustedNodePolicy{}

// ReconcileTrustedNodePolicy reconciles a TrustedNodePolicy object
type ReconcileTrustedNodePolicy struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client   client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
}

// Reconcile reads that state of the cluster for a TrustedNodePolicy object and makes changes based on the state read
// and what is in the TrustedNodePolicy.Spec
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTrustedNodePolicy) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TrustedNodePolicy")

	// Fetch the TrustedNodePolicy instance
	instance := &policiesv1alpha1.TrustedNodePolicy{}
	if reconcilingAgent == nil {
		reconcilingAgent = r
	}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// name of our mcm custom finalizer
	myFinalizerName := Finalizer

	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		updateNeeded := false
		// The object is not being deleted, so if it might not have our finalizer,
		// then lets add the finalizer and update the object.
		if !containsString(instance.ObjectMeta.Finalizers, myFinalizerName) {
			instance.ObjectMeta.Finalizers = append(instance.ObjectMeta.Finalizers, myFinalizerName)
			updateNeeded = true
		}
		if !ensureDefaultLabel(instance) {
			updateNeeded = true
		}
		if updateNeeded {
			if err := r.client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}
		instance.Status.CompliancyDetails = nil //reset CompliancyDetails
		err := handleAddingPolicy(instance)
		if err != nil {
			glog.V(3).Infof("Failed to handleAddingPolicy")
		}
	} else {
		handleRemovingPolicy(instance)
		// The object is being deleted
		if containsString(instance.ObjectMeta.Finalizers, myFinalizerName) {
			// our finalizer is present, so lets handle our external dependency
			if err := r.deleteExternalDependency(instance); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, err
			}

			// remove our finalizer from the list and update it.
			instance.ObjectMeta.Finalizers = removeString(instance.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.client.Update(context.Background(), instance); err != nil {
				return reconcile.Result{Requeue: true}, nil
			}
		}
		// Our finalizer has finished, so the reconciler can do nothing.
		return reconcile.Result{}, nil
	}
	glog.V(3).Infof("reason: successful processing, subject: policy/%v, namespace: %v, according to policy: %v, additional-info: none",
		instance.Name, instance.Namespace, instance.Name)

	// Pod already exists - don't requeue
	// reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

// PeriodicallyExecSamplePolicies always check status
func PeriodicallyExecSamplePolicies(freq uint) {
	var plcToUpdateMap map[string]*policiesv1alpha1.TrustedNodePolicy
	for {
        var removeNodes []corev1.Node
		fmt.Println("LUMJJB Start loop")
		start := time.Now()
		printMap(availablePolicies.PolicyMap)
		plcToUpdateMap = make(map[string]*policiesv1alpha1.TrustedNodePolicy)
		for namespace, policy := range availablePolicies.PolicyMap {
			//For each namespace, fetch all the RoleBindings in that NS according to the policy selector
			//For each RoleBindings get the number of users
			//update the status internal map
			//no difference between enforce and inform here
			// XXX(LUMJJB): Get node list
			nodeList, err := (*common.KubeClient).CoreV1().Nodes().
				List(metav1.ListOptions{LabelSelector: "trusted=false"})
			if err != nil {
				glog.Errorf("reason: communication error, subject: k8s API server, namespace: %v, according to policy: %v, additional-info: %v\n",
					namespace, policy.Name, err)
				continue
			}

			// XXX(LUMJJB): Check trust on node to see if violation
			nodeViolations := checkViolations(nodeList, policy)
			fmt.Printf("Node violations: %s\n", nodeViolations)

			if strings.EqualFold(string(policy.Spec.RemediationAction), string(policiesv1alpha1.Enforce)) {
				if addEnforcement(policy, nodeViolations, namespace) {
                    plcToUpdateMap[policy.Name] = policy
                    removeNodes = append(removeNodes, nodeList.Items...)
                }
			} else {
				if addViolationIfExists(policy, nodeViolations, namespace) {
					plcToUpdateMap[policy.Name] = policy
				}
			}
		}

		//update status of all policies that changed:
		faultyPlc, err := updatePolicyStatus(plcToUpdateMap)
		if err != nil {
			glog.Errorf("reason: policy update error, subject: policy/%v, namespace: %v, according to policy: %v, additional-info: %v\n",
				faultyPlc.Name, faultyPlc.Namespace, faultyPlc.Name, err)
		}

		// Do enforcement
		for _, n := range removeNodes {
			n.Labels["trusted"] = "removed"
			_, err := (*common.KubeClient).CoreV1().Nodes().Update(&n)
			if err != nil {
                glog.Errorf("Unable to update node %v: %v", n.Name, err)
			}
		}

		// making sure that if processing is > freq we don't sleep
		// if freq > processing we sleep for the remaining duration
		elapsed := time.Since(start) / 1000000000 // convert to seconds
		if float64(freq) > float64(elapsed) {
			remainingSleep := float64(freq) - float64(elapsed)
			time.Sleep(time.Duration(remainingSleep) * time.Second)
		}
		if KubeClient == nil {
			return
		}
	}
}

func ensureDefaultLabel(instance *policiesv1alpha1.TrustedNodePolicy) (updateNeeded bool) {
	//we need to ensure this label exists -> category: "System and Information Integrity"
	if instance.ObjectMeta.Labels == nil {
		newlbl := make(map[string]string)
		newlbl["category"] = grcCategory
		instance.ObjectMeta.Labels = newlbl
		return true
	}
	if _, ok := instance.ObjectMeta.Labels["category"]; !ok {
		instance.ObjectMeta.Labels["category"] = grcCategory
		return true
	}
	if instance.ObjectMeta.Labels["category"] != grcCategory {
		instance.ObjectMeta.Labels["category"] = grcCategory
		return true
	}
	return false
}

func checkAllClusterLevel(clusterRoleBindingList *v1.ClusterRoleBindingList) (userV, groupV int) {
	usersMap := make(map[string]bool)
	groupsMap := make(map[string]bool)
	for _, clusterRoleBinding := range clusterRoleBindingList.Items {
		for _, subject := range clusterRoleBinding.Subjects {
			if subject.Kind == "User" {
				usersMap[subject.Name] = true
			}
			if subject.Kind == "Group" {
				groupsMap[subject.Name] = true
			}
		}
	}
	return len(usersMap), len(groupsMap)
}

func convertMaptoPolicyNameKey() map[string]*policiesv1alpha1.TrustedNodePolicy {
	plcMap := make(map[string]*policiesv1alpha1.TrustedNodePolicy)
	for _, policy := range availablePolicies.PolicyMap {
		plcMap[policy.Name] = policy
	}
	return plcMap
}

func checkViolations(nodeList *corev1.NodeList, plc *policiesv1alpha1.TrustedNodePolicy) []string {
	nodes := []string{}
	for _, n := range nodeList.Items {
		nodes = append(nodes, n.Name)
	}
	return nodes
}

func addEnforcement(plc *policiesv1alpha1.TrustedNodePolicy, nodeViolations []string, namespace string) bool {

    if len(nodeViolations) == 0 {
        return false
    }

	if plc.Status.CompliancyDetails == nil {
		plc.Status.CompliancyDetails = make(map[string]map[string][]string)
	}
	if _, ok := plc.Status.CompliancyDetails[plc.Name]; !ok {
		plc.Status.CompliancyDetails[plc.Name] = make(map[string][]string)
	}
	if plc.Status.CompliancyDetails[plc.Name][namespace] == nil {
		plc.Status.CompliancyDetails[plc.Name][namespace] = []string{}
	}
	if len(plc.Status.CompliancyDetails[plc.Name][namespace]) == 0 {
		plc.Status.CompliancyDetails[plc.Name][namespace] = []string{""}
	}

    plc.Status.ComplianceState = policiesv1alpha1.Compliant
    msg := fmt.Sprintf("Compliant after removing untrusted nodes: %v", nodeViolations)
    plc.Status.CompliancyDetails[plc.Name][namespace][0] = msg
    fmt.Println(msg)
    return true
}

/*
func checkComplianceChangeBasedOnDetails(plc *policiesv1alpha1.TrustedNodePolicy) (complianceChanged bool) {
	//used in case we also want to know not just the compliance state, but also whether the compliance changed or not.
	previous := plc.Status.ComplianceState
	if plc.Status.CompliancyDetails == nil {
		plc.Status.ComplianceState = policiesv1alpha1.UnknownCompliancy
		return reflect.DeepEqual(previous, plc.Status.ComplianceState)
	}
	if _, ok := plc.Status.CompliancyDetails[plc.Name]; !ok {
		plc.Status.ComplianceState = policiesv1alpha1.UnknownCompliancy
		return reflect.DeepEqual(previous, plc.Status.ComplianceState)
	}
	if len(plc.Status.CompliancyDetails[plc.Name]) == 0 {
		plc.Status.ComplianceState = policiesv1alpha1.UnknownCompliancy
		return reflect.DeepEqual(previous, plc.Status.ComplianceState)
	}
	plc.Status.ComplianceState = policiesv1alpha1.Compliant
	for namespace, msgList := range plc.Status.CompliancyDetails[plc.Name] {
		if len(msgList) > 0 {
			violationNum := strings.Split(plc.Status.CompliancyDetails[plc.Name][namespace][0], " ")
			if len(violationNum) > 0 {
				if violationNum[0] != fmt.Sprint(0) {
					plc.Status.ComplianceState = policiesv1alpha1.NonCompliant
				}
			}
		} else {
			return reflect.DeepEqual(previous, plc.Status.ComplianceState)
		}
	}
	if plc.Status.ComplianceState != policiesv1alpha1.NonCompliant {
		plc.Status.ComplianceState = policiesv1alpha1.Compliant
	}
	return reflect.DeepEqual(previous, plc.Status.ComplianceState)
}
*/

func addViolationIfExists(plc *policiesv1alpha1.TrustedNodePolicy, nodeViolations []string, namespace string) bool {
	if plc.Status.CompliancyDetails == nil {
		plc.Status.CompliancyDetails = make(map[string]map[string][]string)
	}
	if _, ok := plc.Status.CompliancyDetails[plc.Name]; !ok {
		plc.Status.CompliancyDetails[plc.Name] = make(map[string][]string)
	}
	if plc.Status.CompliancyDetails[plc.Name][namespace] == nil {
		plc.Status.CompliancyDetails[plc.Name][namespace] = []string{}
	}
	if len(plc.Status.CompliancyDetails[plc.Name][namespace]) == 0 {
		plc.Status.CompliancyDetails[plc.Name][namespace] = []string{""}
	}

	if len(nodeViolations) > 0 {
		plc.Status.ComplianceState = policiesv1alpha1.NonCompliant
		msg := fmt.Sprintf("%d violations detected, nodes are: %v",
			len(nodeViolations),
			nodeViolations)
		if plc.Status.CompliancyDetails[plc.Name][namespace][0] == msg {
			return false
		}

		plc.Status.CompliancyDetails[plc.Name][namespace][0] = msg
		fmt.Println(msg)
		return true

	} else {
		// Case where it was compliant and still compliant, preserve message
		if plc.Status.ComplianceState == policiesv1alpha1.Compliant {
			// preserve message
			return false
		}

		msg := "All nodes are trusted"
		plc.Status.ComplianceState = policiesv1alpha1.Compliant
		plc.Status.CompliancyDetails[plc.Name][namespace][0] = msg
		fmt.Println(msg)
		return true
	}
}

/*
func checkComplianceChangeBasedOnDetails(plc *policiesv1alpha1.TrustedNodePolicy) (complianceChanged bool) {
	//used in case we also want to know not just the compliance state, but also whether the compliance changed or not.
	previous := plc.Status.ComplianceState
	if plc.Status.CompliancyDetails == nil {
		plc.Status.ComplianceState = policiesv1alpha1.UnknownCompliancy
		return reflect.DeepEqual(previous, plc.Status.ComplianceState)
	}
	if _, ok := plc.Status.CompliancyDetails[plc.Name]; !ok {
		plc.Status.ComplianceState = policiesv1alpha1.UnknownCompliancy
		return reflect.DeepEqual(previous, plc.Status.ComplianceState)
	}
	if len(plc.Status.CompliancyDetails[plc.Name]) == 0 {
		plc.Status.ComplianceState = policiesv1alpha1.UnknownCompliancy
		return reflect.DeepEqual(previous, plc.Status.ComplianceState)
	}
	plc.Status.ComplianceState = policiesv1alpha1.Compliant
	for namespace, msgList := range plc.Status.CompliancyDetails[plc.Name] {
		if len(msgList) > 0 {
			violationNum := strings.Split(plc.Status.CompliancyDetails[plc.Name][namespace][0], " ")
			if len(violationNum) > 0 {
				if violationNum[0] != fmt.Sprint(0) {
					plc.Status.ComplianceState = policiesv1alpha1.NonCompliant
				}
			}
		} else {
			return reflect.DeepEqual(previous, plc.Status.ComplianceState)
		}
	}
	if plc.Status.ComplianceState != policiesv1alpha1.NonCompliant {
		plc.Status.ComplianceState = policiesv1alpha1.Compliant
	}
	return reflect.DeepEqual(previous, plc.Status.ComplianceState)
}
*/

func updatePolicyStatus(policies map[string]*policiesv1alpha1.TrustedNodePolicy) (*policiesv1alpha1.TrustedNodePolicy, error) {
	for _, instance := range policies { // policies is a map where: key = plc.Name, value = pointer to plc
		err := reconcilingAgent.client.Status().Update(context.TODO(), instance)
		if err != nil {
			return instance, err
		}
		if EventOnParent != "no" {
			createParentPolicyEvent(instance)
		}
		if reconcilingAgent.recorder != nil {
			reconcilingAgent.recorder.Event(instance, "Normal", "Policy updated", fmt.Sprintf("Policy status is: %v", instance.Status.ComplianceState))
		}
	}
	return nil, nil
}

func getContainerID(pod corev1.Pod, containerName string) string {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.Name == containerName {
			return containerStatus.ContainerID
		}
	}
	return ""
}

func handleRemovingPolicy(plc *policiesv1alpha1.TrustedNodePolicy) {
	for k, v := range availablePolicies.PolicyMap {
		if v.Name == plc.Name {
			availablePolicies.RemoveObject(k)
		}
	}
}

func handleAddingPolicy(plc *policiesv1alpha1.TrustedNodePolicy) error {
	allNamespaces, err := common.GetAllNamespaces()
	if err != nil {
		glog.Errorf("reason: error fetching the list of available namespaces, subject: K8s API server, namespace: all, according to policy: %v, additional-info: %v",
			plc.Name, err)
		return err
	}
	//clean up that policy from the existing namepsaces, in case the modification is in the namespace selector
	for _, ns := range allNamespaces {
		if policy, found := availablePolicies.GetObject(ns); found {
			if policy.Name == plc.Name {
				availablePolicies.RemoveObject(ns)
			}
		}
	}
	selectedNamespaces := common.GetSelectedNamespaces(plc.Spec.NamespaceSelector.Include, plc.Spec.NamespaceSelector.Exclude, allNamespaces)
	for _, ns := range selectedNamespaces {
		availablePolicies.AddObject(ns, plc)
	}
	return err
}

//=================================================================
//deleteExternalDependency in case the CRD was related to non-k8s resource
//nolint
func (r *ReconcileTrustedNodePolicy) deleteExternalDependency(instance *policiesv1alpha1.TrustedNodePolicy) error {
	glog.V(0).Infof("reason: CRD deletion, subject: policy/%v, namespace: %v, according to policy: none, additional-info: none\n",
		instance.Name,
		instance.Namespace)
	// Ensure that delete implementation is idempotent and safe to invoke
	// multiple types for same object.
	return nil
}

//=================================================================
// Helper functions to check if a string exists in a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

//=================================================================
// Helper functions to remove a string from a slice of strings.
func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}

//=================================================================
// Helper functions that pretty prints a map
func printMap(myMap map[string]*policiesv1alpha1.TrustedNodePolicy) {
	if len(myMap) == 0 {
		fmt.Println("Waiting for policies to be available for processing... ")
		return
	}
	fmt.Println("Available policies in namespaces: ")

	for k, v := range myMap {
		fmt.Printf("namespace = %v; policy = %v \n", k, v.Name)
	}
}

func createParentPolicyEvent(instance *policiesv1alpha1.TrustedNodePolicy) {
	if len(instance.OwnerReferences) == 0 {
		return //there is nothing to do, since no owner is set
	}
	// we are making an assumption that the GRC policy has a single owner, or we chose the first owner in the list
	if string(instance.OwnerReferences[0].UID) == "" {
		return //there is nothing to do, since no owner UID is set
	}

	parentPlc := createParentPolicy(instance)

	reconcilingAgent.recorder.Event(&parentPlc,
		corev1.EventTypeNormal,
		fmt.Sprintf("policy: %s/%s", instance.Namespace, instance.Name),
		convertPolicyStatusToString(instance))
}

func createParentPolicy(instance *policiesv1alpha1.TrustedNodePolicy) policiesv1alpha1.Policy {
	ns := common.ExtractNamespaceLabel(instance)
	if ns == "" {
		ns = NamespaceWatched
	}
	plc := policiesv1alpha1.Policy{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.OwnerReferences[0].Name,
			Namespace: ns, // we are making an assumption here that the parent policy is in the watched-namespace passed as flag
			UID:       instance.OwnerReferences[0].UID,
		},
		TypeMeta: metav1.TypeMeta{
			Kind:       "Policy",
			APIVersion: " policies.ibm.com/v1alpha1",
		},
	}
	return plc
}

//=================================================================
// convertPolicyStatusToString to be able to pass the status as event
func convertPolicyStatusToString(plc *policiesv1alpha1.TrustedNodePolicy) (results string) {
	result := "ComplianceState is still undetermined"
	if plc.Status.ComplianceState == "" {
		return result
	}
	result = string(plc.Status.ComplianceState)

	if plc.Status.CompliancyDetails == nil {
		return result
	}
	if _, ok := plc.Status.CompliancyDetails[plc.Name]; !ok {
		return result
	}
	for _, v := range plc.Status.CompliancyDetails[plc.Name] {
		result += fmt.Sprintf("; %s", strings.Join(v, ", "))
	}
	return result
}
