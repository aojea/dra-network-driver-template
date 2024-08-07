package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"time"

	resourceapi "k8s.io/api/resource/v1alpha3"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/dynamic-resource-allocation/kubeletplugin"
	"k8s.io/klog/v2"
)

const (
	// DEFINE THE DRIVER NAME
	driverName = "dra.networking.driver"
	// DEFINE THE PERIOD THE DRIVER DISCOVER THE LOCAL RESOURCES
	discoveryPeriod = 1 * time.Minute
)

// DISCOVER LOCAL RESOURCES TO BE PUBLISHED ON THE ResourceSlice
func discoverResources(ctx context.Context) kubeletplugin.Resources {
	resources := kubeletplugin.Resources{}

	// ---------------------------------------------------------------------------
	// EXAMPLE: publish all local devices that are not loopback or veth interfaces
	// ---------------------------------------------------------------------------
	ifaces, err := net.Interfaces()
	if err != nil {
		klog.Infof("error getting system interfaces: %v", err)
	}

	for _, iface := range ifaces {
		// Create the basic Device
		device := resourceapi.Device{
			Name: iface.Name,
			Basic: &resourceapi.BasicDevice{
				Attributes: make(map[resourceapi.QualifiedName]resourceapi.DeviceAttribute),
				Capacity:   make(map[resourceapi.QualifiedName]resource.Quantity),
			},
		}
		// Add Attributes that can be used as selectors
		mac := iface.HardwareAddr.String()
		device.Basic.Attributes["mac"] = resourceapi.DeviceAttribute{StringValue: &mac}
		mtu := int64(iface.MTU)
		device.Basic.Attributes["mtu"] = resourceapi.DeviceAttribute{IntValue: &mtu}
		device.Basic.Attributes["name"] = resourceapi.DeviceAttribute{StringValue: &iface.Name}
		// Use only the first IP if set
		if ips, err := iface.Addrs(); err == nil && len(ips) > 0 {
			ip := ips[0].String()
			device.Basic.Attributes["ip"] = resourceapi.DeviceAttribute{StringValue: &ip}
		}
		// Add Capacity that can be used as selectors
		// device.Basic.Capacity["speed"] = *resource.NewQuantity(1000, resource.DecimalSI)

		// append to existing devices
		resources.Devices = append(resources.Devices, device)
	}

	return resources
}

// POD LIFECYCLE HOOKS

// Pod Start runs AFTER CNI ADD and BEFORE the containers are created
// It runs in the Container Runtime network namespace and receives as paremeters:
// - the Pod network namespace
// - the ResourceClaim AllocationResult
func podStartHook(ctx context.Context, netns string, allocation resourceapi.AllocationResult) error {
	// Process the configurations of the ResourceClaim
	for _, config := range allocation.Devices.Config {
		if config.Opaque == nil {
			continue
		}
		klog.V(4).Infof("podStartHook Configuration %s", config.Opaque.Parameters.String())
		// TODO get config options here, it can add ips or commands
		// to add routes, run dhcp, rename the interface ... whatever
	}
	// Process the configurations of the ResourceClaim
	for _, result := range allocation.Devices.Results {
		if result.Driver != driverName {
			continue
		}
		klog.V(4).Infof("podStartHook Device %s", result.Device)
		// ------------------------------------------------------------------
		// EXAMPLE: move interface by name to the specified network namespace
		// ------------------------------------------------------------------
		// TODO see https://github.com/containernetworking/plugins/tree/main/plugins/main
		// for better examples of low level implementations using netlink for more complex
		// scenarios like host-device, ipvlan, macvlan, ...
		cmd := exec.Command("ip", "link", "set", result.Device, "netns", netns)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to move interface %s to namespace %s: %w", result.Device, netns, err)
		}
	}
	return nil
}

// Pod Stop runs on Pod deletion, Pod deletion shoud be best effort, is recommended
// to avoid returning an error in this hook.
// It runs in the Container Runtime network namespace and receives as paremeters:
// - the Pod network namespace
// - the ResourceClaim allocation
func podStopHook(ctx context.Context, netns string, allocation resourceapi.AllocationResult) error {
	// Process the configurations of the ResourceClaim
	for _, config := range allocation.Devices.Config {
		if config.Opaque == nil {
			continue
		}
		klog.V(4).Infof("podStopHook Configuration %s", config.Opaque.Parameters.String())
		// TODO get config options here, it can add ips or commands
		// to add routes, run dhcp, rename the interface ... whatever
	}
	// Process the configurations of the ResourceClaim
	for _, result := range allocation.Devices.Results {
		if result.Driver != driverName {
			continue
		}
		klog.V(4).Infof("podStopHook Device %s", result.Device)
		// TODO get config options here, it can add ips or commands
		// to add routes, run dhcp, rename the interface ... whatever
	}
	return nil
}
