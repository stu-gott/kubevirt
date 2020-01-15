package webhooks

import (
	"fmt"

	"k8s.io/api/admission/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	webhookutils "kubevirt.io/kubevirt/pkg/util/webhooks"
	validating_webhooks "kubevirt.io/kubevirt/pkg/util/webhooks/validating-webhooks"
)

const uninstallErrorMsg = "Rejecting the uninstall request, since there are still %s present. Either delete all KubeVirt related workloads or change the uninstall strategy before uninstalling KubeVirt."

var KubeVirtGroupVersionResource = metav1.GroupVersionResource{
	Group:    v1.VirtualMachineInstanceGroupVersionKind.Group,
	Version:  v1.VirtualMachineInstanceGroupVersionKind.Version,
	Resource: "kubevirts",
}

func NewKubeVirtDeletionAdmitter(client kubecli.KubevirtClient) *KubeVirtDeletionAdmitter {
	return &KubeVirtDeletionAdmitter{
		client: client,
	}
}

type KubeVirtDeletionAdmitter struct {
	client kubecli.KubevirtClient
}

func (k *KubeVirtDeletionAdmitter) Admit(review *v1beta1.AdmissionReview) *v1beta1.AdmissionResponse {
	obj, err := k.client.KubeVirt(review.Request.Namespace).Get(review.Request.Name, &metav1.GetOptions{})
	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if obj.Spec.UninstallStrategy == "" || obj.Spec.UninstallStrategy == v1.KubeVirtUninstallStrategyRemoveWorkloads {
		return validating_webhooks.NewPassingAdmissionResponse()
	}

	vmis, err := k.client.VirtualMachineInstance(metav1.NamespaceAll).List(&metav1.ListOptions{Limit: 2})

	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(vmis.Items) > 0 {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf(uninstallErrorMsg, "Virtual Machine Instances"))
	}

	vms, err := k.client.VirtualMachine(metav1.NamespaceAll).List(&metav1.ListOptions{Limit: 2})

	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(vms.Items) > 0 {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf(uninstallErrorMsg, "Virtual Machines"))
	}

	vmirs, err := k.client.ReplicaSet(metav1.NamespaceAll).List(metav1.ListOptions{Limit: 2})

	if err != nil {
		return webhookutils.ToAdmissionResponseError(err)
	}

	if len(vmirs.Items) > 0 {
		return webhookutils.ToAdmissionResponseError(fmt.Errorf(uninstallErrorMsg, "Virtual Machine Instance Replica Sets"))
	}

	return validating_webhooks.NewPassingAdmissionResponse()
}
