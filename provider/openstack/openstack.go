/*
 * Copyright (C) 2015 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package openstack

import (
	"fmt"
	"math"
	"os"

	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/openstack/imageservice/v2/imagedata"
	glanceImages "github.com/gophercloud/gophercloud/openstack/imageservice/v2/images"
	"github.com/gophercloud/gophercloud/pagination"
)

// OpenStackComputeManager provides OpenStack Nova API interaction.
type OpenStackComputeManager interface {
	// ListFlavors returns list of flavors matching disk and mem criteria.
	// Args:
	// - minDiskGB   ... only list flavors with disk >= minDiskGB
	// - minMemoryMB ... only list flavors with mem >= minMemoryMB
	ListFlavors(int, int) []flavors.Flavor

	// LaunchServer launches single server of given image.
	// Args:
	// - name       ... new server name
	// - flavorName ... flavor to launch server with
	// - imageName  ... image to base server on
	// - verbose
	LaunchServer(string, string, string, bool) error

	// GetFlavor returns flavors.Flavor struct for given flavorName.
	// Args:
	// - flavorName ... flavor name e.g. m1.tiny
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

// OpenStackImageManager provides OpenStack Glance API interaction.
type OpenStackImageManager interface {
	// CreateImage creates image metadata on OpenStack.
	// Args:
	// - name    ... new image name
	// - flavor  ... flavor to fit image size to
	// - verbose
	// Returns:
	// - imageId ... id of created image
	CreateImage(string, *flavors.Flavor, bool) (string, error)

	// UploadImage uploads image binary data to existing OpenStack image metadata.
	// Args:
	// - imageId  ... image metadata id returned by CreateImage
	// - filepath ... path to .qcow2 image file
	// - verbose
	UploadImage(string, string, bool) error

	// DeleteImage deletes image from OpenStack.
	// Args:
	// - imageId ... image metadata id returned by CreateImage
	DeleteImage(string)
}

// GetOrPickFlavor returns flavors.Flavor struct that best matches arguments.
// First it decides whether user wanted to get flavor by name or by criteria.
// In case of flavor name, it retrieves data about that flavor and returns it.
// In case of criteria (diskMB, memoryMB) it returns smallest flavor that meets all criteria.
func GetOrPickFlavor(m OpenStackComputeManager, flavorName string, diskMB int64, memoryMB int64, verbose bool) (*flavors.Flavor, error) {
	if flavorName != "" {
		return m.GetFlavor(flavorName, verbose)
	}
	return PickFlavor(m, diskMB, memoryMB, verbose)
}

// PickFlavor picks flavor that best matches criteria (i.e. HDD size and RAM size).
// While diskMB is required, memoryMB is optional (set to -1 to ignore).
func PickFlavor(m OpenStackComputeManager, diskMB int64, memoryMB int64, verbose bool) (*flavors.Flavor, error) {
	if diskMB <= 0 {
		return nil, fmt.Errorf("Please specify disk size.")
	}

	var flavs []flavors.Flavor = m.ListFlavors(int(math.Ceil(float64(diskMB)/1024)), int(memoryMB))
	return selectBestFlavor(flavs, verbose)
}

// PushImage first creates meta for image at OpenStack, then it sends binary data for it, the qcow2 image.
func PushImage(m OpenStackImageManager, imageName string, imageFilepath string, flavor *flavors.Flavor, verbose bool) {
	// Create metadata (on OpenStack).
	imgId, _ := m.CreateImage(imageName, flavor, verbose)
	// Send the image binary data to OpenStack
	m.UploadImage(imgId, imageFilepath, verbose)
}

// LaunchInstances launches <count> instances. Return first error that occurs or nil on success.
func LaunchInstances(m OpenStackComputeManager, name string, imageName string, flavorName string, count int, verbose bool) error {
	var err error
	if count <= 1 {
		// Take name as it is.
		err = m.LaunchServer(name, flavorName, imageName, verbose)
	} else {
		// Append index after the name of each instance.
		for i := 0; i < count; i++ {
			currErr := m.LaunchServer(fmt.Sprintf("%s-%d", name, (i+1)), flavorName, imageName, verbose)
			if err == nil {
				err = currErr
			}
		}
	}

	return err
}

// openStackManager implements both OpenStackComputeManager and OpenStackImageManager interface.
type openStackManager struct {
	clientNova   *gophercloud.ServiceClient
	clientGlance *gophercloud.ServiceClient
}

func (m *openStackManager) ListFlavors(minDiskGB int, minMemoryMB int) []flavors.Flavor {
	var flavs []flavors.Flavor = make([]flavors.Flavor, 0)

	pagerFlavors := flavors.ListDetail(m.clientNova, flavors.ListOpts{
		MinDisk: minDiskGB,
		MinRAM:  minMemoryMB,
	})
	pagerFlavors.EachPage(func(page pagination.Page) (bool, error) {
		flavorList, _ := flavors.ExtractFlavors(page)

		for _, f := range flavorList {
			flavs = append(flavs, f)
		}

		return true, nil
	})
	return flavs
}

func (m *openStackManager) LaunchServer(name string, flavorName string, imageName string, verbose bool) error {
	resp := servers.Create(m.clientNova, servers.CreateOpts{
		Name:          name,
		FlavorName:    flavorName,
		ImageName:     imageName,
		ServiceClient: m.clientNova, // need to pass this to perform name-to-ID lookup
	})
	if verbose {
		instance, err := resp.Extract()
		if err != nil {
			fmt.Println("Unable to create instance:", err)
		} else {
			fmt.Printf("Instance ID: %s\n", instance.ID)
		}
	}

	return resp.Err
}

func (m *openStackManager) CreateImage(name string, flavor *flavors.Flavor, verbose bool) (string, error) {
	createdImage, err := glanceImages.Create(m.clientGlance, glanceImages.CreateOpts{
		Name:            name,
		Tags:            []string{"tagOSv", "tagCapstan"},
		DiskFormat:      "qcow2",
		ContainerFormat: "bare",
		MinDisk:         flavor.Disk,
	}).Extract()

	if err == nil && verbose {
		fmt.Printf("Created image [name: %s, ID: %s]\n", createdImage.Name, createdImage.ID)
	}
	return createdImage.ID, err
}

func (m *openStackManager) UploadImage(imageId string, filepath string, verbose bool) error {
	if verbose {
		fmt.Printf("Uploading composed image '%s' to OpenStack\n", filepath)
	}

	f, err := os.Open(filepath)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	res := imagedata.Upload(m.clientGlance, imageId, f)
	return res.Err
}

func (m *openStackManager) DeleteImage(imageId string) {
	glanceImages.Delete(m.clientGlance, imageId)
}

func (m *openStackManager) GetFlavor(flavorName string, verbose bool) (*flavors.Flavor, error) {
	flavorId, err := flavors.IDFromName(m.clientNova, flavorName)
	if err != nil {
		return nil, err
	}

	flavor, err := flavors.Get(m.clientNova, flavorId).Extract()
	if err != nil {
		return nil, err
	}

	return flavor, nil
}

func (m *openStackManager) GetImageMeta(imageName string, verbose bool) (*images.Image, error) {
	imageId, err := images.IDFromName(m.clientNova, imageName)
	if err != nil {
		return nil, err
	}

	image, err := images.Get(m.clientNova, imageId).Extract()
	if err != nil {
		return nil, err
	}

	return image, nil
}

// selectBestFlavor selects optimal flavor for given conditions.
// Argument matchingFlavors should be a list of flavors that all satisfy disk&ram conditions.
func selectBestFlavor(matchingFlavors []flavors.Flavor, verbose bool) (*flavors.Flavor, error) {
	if len(matchingFlavors) == 0 {
		return nil, fmt.Errorf("No matching flavors to pick from")
	}

	var bestFlavor *flavors.Flavor
	for _, f := range matchingFlavors {
		curr := f
		// Prefer better fitting of HDD size rather than RAM size.
		if bestFlavor == nil ||
			f.Disk < bestFlavor.Disk ||
			(f.Disk == bestFlavor.Disk && f.RAM < bestFlavor.RAM) {
			bestFlavor = &curr
		}
	}
	return bestFlavor, nil
}
