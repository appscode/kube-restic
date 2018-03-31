package v1alpha1

import (
	crdutils "github.com/appscode/kutil/apiextensions/v1beta1"
	apiextensions "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
)

func (c Restic) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Version:       SchemeGroupVersion.Version,
		Plural:        ResourcePluralRestic,
		Singular:      ResourceSingularRestic,
		Kind:          ResourceKindRestic,
		ListKind:      ResourceKindRestic + "List",
		ShortNames:    []string{"rst"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "stash"},
		},
		SpecDefinitionName:    "github.com/appscode/stash/apis/stash/v1alpha1.Restic",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitions,
	})
}

func (c Recovery) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Version:       SchemeGroupVersion.Version,
		Plural:        ResourcePluralRecovery,
		Singular:      ResourceSingularRecovery,
		Kind:          ResourceKindRecovery,
		ListKind:      ResourceKindRecovery + "List",
		ShortNames:    []string{"rec"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "stash"},
		},
		SpecDefinitionName:    "github.com/appscode/stash/apis/stash/v1alpha1.Recovery",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitions,
	})
}

func (c Repository) CustomResourceDefinition() *apiextensions.CustomResourceDefinition {
	return crdutils.NewCustomResourceDefinition(crdutils.Config{
		Group:         SchemeGroupVersion.Group,
		Version:       SchemeGroupVersion.Version,
		Plural:        ResourcePluralRepository,
		Singular:      ResourceSingularRepository,
		Kind:          ResourceKindRepository,
		ListKind:      ResourceKindRepository + "List",
		ShortNames:    []string{"repo"},
		ResourceScope: string(apiextensions.NamespaceScoped),
		Labels: crdutils.Labels{
			LabelsMap: map[string]string{"app": "stash"},
		},
		SpecDefinitionName:    "github.com/appscode/stash/apis/stash/v1alpha1.Repository",
		EnableValidation:      true,
		GetOpenAPIDefinitions: GetOpenAPIDefinitions,
	})
}
