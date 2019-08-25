package tests_test

import (
	"encoding/xml"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	k8sv1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "kubevirt.io/client-go/api/v1"
	"kubevirt.io/client-go/kubecli"
	"kubevirt.io/client-go/log"
	"kubevirt.io/kubevirt/pkg/util"
	"kubevirt.io/kubevirt/pkg/virt-launcher/virtwrap/api"

	"kubevirt.io/kubevirt/tests"
)

func parseDeviceAddress(addrString string) []string {
	addrs := strings.Split(addrString, ",")
	naddrs := len(addrs)
	if naddrs > 0 {
		if addrs[naddrs-1] == "" {
			addrs = addrs[:naddrs-1]
		}
	}

	for index, element := range addrs {
		addrs[index] = strings.TrimSpace(element)
	}
	return addrs
}

var _ = Describe("GPU VirtualMachineInstance", func() {
	tests.FlagParse()

	virtClient, err := kubecli.GetKubevirtClient()
	tests.PanicOnError(err)

	tests.BeforeAll(func() {
		tests.SkipIfNoGpuDevicePlugin(virtClient)
	})

	Context("with ephemeral disk", func() {
		It("Should create a valid VMI but pod should not got to running state", func() {
			gpuName := "random.com/gpu"
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
			gpus := []v1.Gpu{
				v1.Gpu{
					Name: gpuName,
				},
			}
			randomVMI.Spec.Domain.Devices.Gpus = gpus
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())

			pod := tests.GetPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)
			Expect(pod.Status.Phase).To(Equal(k8sv1.PodPending))
			Expect(pod.Status.Conditions[0].Type).To(Equal(k8sv1.PodScheduled))
			Expect(strings.Contains(pod.Status.Conditions[0].Message, "Insufficient "+gpuName)).To(Equal(true))
			Expect(pod.Status.Conditions[0].Reason).To(Equal("Unschedulable"))
		})

		It("Should create a valid VMI and appropriate libvirt domain", func() {
			nodesList, err := virtClient.CoreV1().Nodes().List(metav1.ListOptions{})
			var gpuName = ""
			for _, item := range nodesList.Items {
				resourceList := item.Status.Allocatable
				for k, v := range resourceList {
					if strings.HasPrefix(string(k), "nvidia.com") {
						if v.Value() >= 1 {
							gpuName = string(k)
							break
						}
					}
				}
			}
			Expect(gpuName).ToNot(Equal(""))
			randomVMI := tests.NewRandomVMIWithEphemeralDisk(tests.ContainerDiskFor(tests.ContainerDiskCirros))
			gpus := []v1.Gpu{
				v1.Gpu{
					Name: gpuName,
				},
			}
			randomVMI.Spec.Domain.Devices.Gpus = gpus
			vmi, apiErr := virtClient.VirtualMachineInstance(tests.NamespaceTestDefault).Create(randomVMI)
			Expect(apiErr).ToNot(HaveOccurred())
			tests.WaitForSuccessfulVMIStart(vmi)
			domain, err := tests.GetRunningVirtualMachineInstanceDomainXML(virtClient, vmi)
			Expect(err).ToNot(HaveOccurred())
			domSpec := &api.DomainSpec{}
			Expect(xml.Unmarshal([]byte(domain), domSpec)).To(Succeed())

			readyPod := tests.GetRunningPodByVirtualMachineInstance(vmi, tests.NamespaceTestDefault)

			gpuOutput, err := tests.ExecuteCommandOnPod(
				virtClient,
				readyPod,
				"compute",
				[]string{"printenv", "GPU_PASSTHROUGH_DEVICES"},
			)
			log.Log.Infof("%v", gpuOutput)
			Expect(err).ToNot(HaveOccurred())
			addrList := parseDeviceAddress(gpuOutput)

			Expect(len(addrList)).To(Equal(len(domSpec.Devices.HostDevices)))
			for n, addr := range addrList {
				Expect(domSpec.Devices.HostDevices[n].Type).To(Equal("pci"))
				Expect(domSpec.Devices.HostDevices[n].Managed).To(Equal("yes"))
				Expect(domSpec.Devices.HostDevices[n].Mode).To(Equal("subsystem"))
				dbsfFields, err := util.ParsePciAddress(addr)
				Expect(err).ToNot(HaveOccurred())
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Domain).To(Equal("0x" + dbsfFields[0]))
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Bus).To(Equal("0x" + dbsfFields[1]))
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Slot).To(Equal("0x" + dbsfFields[2]))
				Expect(domSpec.Devices.HostDevices[n].Source.Address.Function).To(Equal("0x" + dbsfFields[3]))
			}
		})
	})
})
