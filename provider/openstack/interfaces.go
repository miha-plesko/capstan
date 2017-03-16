/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package openstack

import (
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
)

type ConnectorI interface {
	// ListFlavors returns list of flavors matching disk and mem criteria.
	// Args:
	// - minDiskGB
	// - minMemoryMB
	ListFlavors(int, int) []flavors.Flavor

	// LaunchServer launches single server of given image.
	// Args:
	// - name
	// - flavorName
	// - imageName
	// - verbose
	LaunchServer(string, string, string, bool) error

	// CreateImage creates image metadata on OpenStack.
	// Args:
	// - name
	// - flavor
	// - verbose
	// Returns:
	// - imageId
	CreateImage(string, *flavors.Flavor, bool) (string, error)

	// UploadImage uploads image binary data to existing OpenStack image metadata.
	// Args:
	// - imageId
	// - filepath
	// - verbose
	UploadImage(string, string, bool) error

	// DeleteImage deletes image from OpenStack.
	// Args:
	// - imageId
	DeleteImage(string)

	// GetFlavor returns flavors.Flavor struct for given flavorName.
	// Args:
	// - flavorName
	// - verbose
	// Returns:
	// - flavor
	GetFlavor(string, bool) (*flavors.Flavor, error)

	// GetImageMeta returns images.Image struct for given imageName.
	// Args:
	// - imageName
	// - verbose
	GetImageMeta(string, bool) (*images.Image, error)
}
