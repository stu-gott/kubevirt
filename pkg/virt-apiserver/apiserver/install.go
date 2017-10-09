package apiserver

import (
	"k8s.io/apimachinery/pkg/apimachinery/announced"
	"k8s.io/apimachinery/pkg/apimachinery/registered"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"

	"kubevirt.io/kubevirt/pkg/api/v1"
)

// Install registers the API group and adds types to a scheme
func Install(groupFactoryRegistry announced.APIGroupFactoryRegistry, registry *registered.APIRegistrationManager, scheme *runtime.Scheme) {
	if err := announced.NewGroupMetaFactory(
		&announced.GroupMetaFactoryArgs{
			GroupName:                  v1.GroupName,
			RootScopedKinds:            sets.NewString("VirtualMachine", "VirtualMachineList", "Migration", "MigrationList", "VirtualMachineReplicaSet", "VirtualMachineReplicaSetList"),
			VersionPreferenceOrder:     []string{v1.SchemeGroupVersion.Version},
			AddInternalObjectsToScheme: v1.AddToScheme,
		},
		announced.VersionToSchemeFunc{
			v1.SchemeGroupVersion.Version: v1.AddToScheme,
		},
	).Announce(groupFactoryRegistry).RegisterAndEnable(registry, scheme); err != nil {
		panic(err)
	}
}
