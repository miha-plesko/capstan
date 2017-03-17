/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package openstack

import (
	"fmt"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
)

type OpenStackComputeManagerMockBase struct {
}

func (m *OpenStackComputeManagerMockBase) ListFlavors(minDiskGB int, minMemoryMB int) []flavors.Flavor {
	return make([]flavors.Flavor, 0)
}

func (m *OpenStackComputeManagerMockBase) LaunchServer(name string, flavorName string, imageName string, verbose bool) error {
	return fmt.Errorf("mock not implemented")
}

func (m *OpenStackComputeManagerMockBase) GetFlavor(flavorName string, verbose bool) (*flavors.Flavor, error) {
	return nil, fmt.Errorf("mock not implemented")
}

func (m *OpenStackComputeManagerMockBase) GetImageMeta(imageName string, verbose bool) (*images.Image, error) {
	return nil, fmt.Errorf("mock not implemented")
}

type OpenStackImageManagerMockBase struct {
}

func (m *OpenStackImageManagerMockBase) CreateImage(name string, flavor *flavors.Flavor, verbose bool) (string, error) {
	return "", fmt.Errorf("mock not implemented")
}

func (m *OpenStackImageManagerMockBase) UploadImage(imageId string, filepath string, verbose bool) error {
	return fmt.Errorf("mock not implemented")
}

func (m *OpenStackImageManagerMockBase) DeleteImage(imageId string) {
}
