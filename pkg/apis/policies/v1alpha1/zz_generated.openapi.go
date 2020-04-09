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
// +build !ignore_autogenerated

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1alpha1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicy":       schema_pkg_apis_policies_v1alpha1_TrustedNodePolicy(ref),
		"github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicySpec":   schema_pkg_apis_policies_v1alpha1_TrustedNodePolicySpec(ref),
		"github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicyStatus": schema_pkg_apis_policies_v1alpha1_TrustedNodePolicyStatus(ref),
	}
}

func schema_pkg_apis_policies_v1alpha1_TrustedNodePolicy(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "TrustedNodePolicy is the Schema for the samplepolicies API",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"kind": {
						SchemaProps: spec.SchemaProps{
							Description: "Kind is a string value representing the REST resource this object represents. Servers may infer this from the endpoint the client submits requests to. Cannot be updated. In CamelCase. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#types-kinds",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"apiVersion": {
						SchemaProps: spec.SchemaProps{
							Description: "APIVersion defines the versioned schema of this representation of an object. Servers should convert recognized schemas to the latest internal value, and may reject unrecognized values. More info: https://git.k8s.io/community/contributors/devel/api-conventions.md#resources",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"metadata": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"),
						},
					},
					"spec": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicySpec"),
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Ref: ref("github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicyStatus"),
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicySpec", "github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.TrustedNodePolicyStatus", "k8s.io/apimachinery/pkg/apis/meta/v1.ObjectMeta"},
	}
}

func schema_pkg_apis_policies_v1alpha1_TrustedNodePolicySpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "TrustedNodePolicySpec defines the desired state of TrustedNodePolicy",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"severity": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
					"remediationAction": {
						SchemaProps: spec.SchemaProps{
							Description: "low, medium, high",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"namespaceSelector": {
						SchemaProps: spec.SchemaProps{
							Description: "enforce, inform",
							Ref:         ref("github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.Target"),
						},
					},
					"labelSelector": {
						SchemaProps: spec.SchemaProps{
							Description: "selecting a list of namespaces where the policy applies",
							Type:        []string{"object"},
							AdditionalProperties: &spec.SchemaOrBool{
								Allows: true,
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"maxRoleBindingUsersPerNamespace": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"maxRoleBindingGroupsPerNamespace": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"maxClusterRoleBindingUsers": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
					"maxClusterRoleBindingGroups": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"integer"},
							Format: "int32",
						},
					},
				},
			},
		},
		Dependencies: []string{
			"github.com/lumjjb/trusted-node-policy-controller/pkg/apis/policies/v1alpha1.Target"},
	}
}

func schema_pkg_apis_policies_v1alpha1_TrustedNodePolicyStatus(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "TrustedNodePolicyStatus defines the observed state of TrustedNodePolicy",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"compliant": {
						SchemaProps: spec.SchemaProps{
							Type:   []string{"string"},
							Format: "",
						},
					},
				},
			},
		},
	}
}
