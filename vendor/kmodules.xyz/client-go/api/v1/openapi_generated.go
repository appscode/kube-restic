// +build !ignore_autogenerated

/*
Copyright AppsCode Inc. and Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by openapi-gen. DO NOT EDIT.

// This file was autogenerated by openapi-gen. Do not edit it manually!

package v1

import (
	spec "github.com/go-openapi/spec"
	common "k8s.io/kube-openapi/pkg/common"
)

func GetOpenAPIDefinitions(ref common.ReferenceCallback) map[string]common.OpenAPIDefinition {
	return map[string]common.OpenAPIDefinition{
		"kmodules.xyz/client-go/api/v1.CertificateSpec": schema_kmodulesxyz_client_go_api_v1_CertificateSpec(ref),
		"kmodules.xyz/client-go/api/v1.Condition":       schema_kmodulesxyz_client_go_api_v1_Condition(ref),
		"kmodules.xyz/client-go/api/v1.TLSConfig":       schema_kmodulesxyz_client_go_api_v1_TLSConfig(ref),
		"kmodules.xyz/client-go/api/v1.X509Subject":     schema_kmodulesxyz_client_go_api_v1_X509Subject(ref),
	}
}

func schema_kmodulesxyz_client_go_api_v1_CertificateSpec(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"alias": {
						SchemaProps: spec.SchemaProps{
							Description: "Alias represents the identifier of the certificate.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"secretName": {
						SchemaProps: spec.SchemaProps{
							Description: "Specifies the k8s secret name that holds the certificates. Default to <resource-name>-<cert-alias>-cert.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"keyEncoding": {
						SchemaProps: spec.SchemaProps{
							Description: "KeyEncoding is the private key cryptography standards (PKCS) for this certificate's private key to be encoded in. If provided, allowed values are \"pkcs1\" and \"pkcs8\". If KeyEncoding is not specified, then PKCS#1 will be used by default.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"subject": {
						SchemaProps: spec.SchemaProps{
							Description: "Full X509 name specification (https://golang.org/pkg/crypto/x509/pkix/#Name).",
							Ref:         ref("kmodules.xyz/client-go/api/v1.X509Subject"),
						},
					},
					"duration": {
						SchemaProps: spec.SchemaProps{
							Description: "Certificate default Duration",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
					"renewBefore": {
						SchemaProps: spec.SchemaProps{
							Description: "Certificate renew before expiration duration",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Duration"),
						},
					},
					"dnsNames": {
						SchemaProps: spec.SchemaProps{
							Description: "DNSNames is a list of subject alt names to be used on the Certificate.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"ipAddresses": {
						SchemaProps: spec.SchemaProps{
							Description: "IPAddresses is a list of IP addresses to be used on the Certificate",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"uriSANs": {
						SchemaProps: spec.SchemaProps{
							Description: "URISANs is a list of URI Subject Alternative Names to be set on this Certificate.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"emailSANs": {
						SchemaProps: spec.SchemaProps{
							Description: "EmailSANs is a list of email subjectAltNames to be set on the Certificate.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
				},
				Required: []string{"alias"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.Duration", "kmodules.xyz/client-go/api/v1.X509Subject"},
	}
}

func schema_kmodulesxyz_client_go_api_v1_Condition(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"type": {
						SchemaProps: spec.SchemaProps{
							Description: "Type of condition in CamelCase or in foo.example.com/CamelCase. Many .condition.type values are consistent across resources like Available, but because arbitrary conditions can be useful (see .node.status.conditions), the ability to deconflict is important.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"status": {
						SchemaProps: spec.SchemaProps{
							Description: "Status of the condition, one of True, False, Unknown.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"observedGeneration": {
						SchemaProps: spec.SchemaProps{
							Description: "If set, this represents the .metadata.generation that the condition was set based upon. For instance, if .metadata.generation is currently 12, but the .status.condition[x].observedGeneration is 9, the condition is out of date with respect to the current state of the instance.",
							Type:        []string{"integer"},
							Format:      "int64",
						},
					},
					"lastTransitionTime": {
						SchemaProps: spec.SchemaProps{
							Description: "Last time the condition transitioned from one status to another. This should be when the underlying condition changed.  If that is not known, then using the time when the API field changed is acceptable.",
							Ref:         ref("k8s.io/apimachinery/pkg/apis/meta/v1.Time"),
						},
					},
					"reason": {
						SchemaProps: spec.SchemaProps{
							Description: "The reason for the condition's last transition in CamelCase. The specific API may choose whether or not this field is considered a guaranteed API. This field may not be empty.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
					"message": {
						SchemaProps: spec.SchemaProps{
							Description: "A human readable message indicating details about the transition. This field may be empty.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
				Required: []string{"type", "status", "lastTransitionTime", "reason", "message"},
			},
		},
		Dependencies: []string{
			"k8s.io/apimachinery/pkg/apis/meta/v1.Time"},
	}
}

func schema_kmodulesxyz_client_go_api_v1_TLSConfig(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Type: []string{"object"},
				Properties: map[string]spec.Schema{
					"issuerRef": {
						SchemaProps: spec.SchemaProps{
							Description: "IssuerRef is a reference to a Certificate Issuer.",
							Ref:         ref("k8s.io/api/core/v1.TypedLocalObjectReference"),
						},
					},
					"certificates": {
						SchemaProps: spec.SchemaProps{
							Description: "Certificate provides server and/or client certificate options used by application pods. These options are passed to a cert-manager Certificate object. xref: https://github.com/jetstack/cert-manager/blob/v0.16.0/pkg/apis/certmanager/v1beta1/types_certificate.go#L82-L162",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Ref: ref("kmodules.xyz/client-go/api/v1.CertificateSpec"),
									},
								},
							},
						},
					},
				},
			},
		},
		Dependencies: []string{
			"k8s.io/api/core/v1.TypedLocalObjectReference", "kmodules.xyz/client-go/api/v1.CertificateSpec"},
	}
}

func schema_kmodulesxyz_client_go_api_v1_X509Subject(ref common.ReferenceCallback) common.OpenAPIDefinition {
	return common.OpenAPIDefinition{
		Schema: spec.Schema{
			SchemaProps: spec.SchemaProps{
				Description: "X509Subject Full X509 name specification",
				Type:        []string{"object"},
				Properties: map[string]spec.Schema{
					"organizations": {
						SchemaProps: spec.SchemaProps{
							Description: "Organizations to be used on the Certificate.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"countries": {
						SchemaProps: spec.SchemaProps{
							Description: "Countries to be used on the CertificateSpec.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"organizationalUnits": {
						SchemaProps: spec.SchemaProps{
							Description: "Organizational Units to be used on the CertificateSpec.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"localities": {
						SchemaProps: spec.SchemaProps{
							Description: "Cities to be used on the CertificateSpec.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"provinces": {
						SchemaProps: spec.SchemaProps{
							Description: "State/Provinces to be used on the CertificateSpec.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"streetAddresses": {
						SchemaProps: spec.SchemaProps{
							Description: "Street addresses to be used on the CertificateSpec.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"postalCodes": {
						SchemaProps: spec.SchemaProps{
							Description: "Postal codes to be used on the CertificateSpec.",
							Type:        []string{"array"},
							Items: &spec.SchemaOrArray{
								Schema: &spec.Schema{
									SchemaProps: spec.SchemaProps{
										Type:   []string{"string"},
										Format: "",
									},
								},
							},
						},
					},
					"serialNumber": {
						SchemaProps: spec.SchemaProps{
							Description: "Serial number to be used on the CertificateSpec.",
							Type:        []string{"string"},
							Format:      "",
						},
					},
				},
			},
		},
	}
}
