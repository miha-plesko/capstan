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

// GetOrPickFlavor returns flavors.Flavor struct that best matches arguments.
// First it decides whether user wanted to get flavor by name or by criteria.
// In case of flavor name, it retrieves data about that flavor and returns it.
// In case of criteria (diskMB, memoryMB) it returns smallest flavor that meets all criteria.
func GetOrPickFlavor(c ConnectorI, flavorName string, diskMB int64, memoryMB int64, verbose bool) (*flavors.Flavor, error) {
	if flavorName != "" {
		return c.GetFlavor(flavorName, verbose)
	}
	return PickFlavor(c, diskMB, memoryMB, verbose)
}

// PickFlavor picks flavor that best matches criteria (i.e. HDD size and RAM size).
// While diskMB is required, memoryMB is optional (set to -1 to ignore).
func PickFlavor(c ConnectorI, diskMB int64, memoryMB int64, verbose bool) (*flavors.Flavor, error) {
	if diskMB <= 0 {
		return nil, fmt.Errorf("Please specify disk size.")
	}

	var flavs []flavors.Flavor = c.ListFlavors(int(math.Ceil(float64(diskMB)/1024)), int(memoryMB))
	return selectBestFlavor(flavs, verbose)
}

// PushImage first creates meta for image at OpenStack, then it sends binary data for it, the qcow2 image.
func PushImage(c ConnectorI, imageName string, imageFilepath string, flavor *flavors.Flavor, verbose bool) {
	// Create metadata (on OpenStack).
	imgId, _ := c.CreateImage(imageName, flavor, verbose)
	// Send the image binary data to OpenStack
	c.UploadImage(imgId, imageFilepath, verbose)
}

// LaunchInstances launches <count> instances. Return first error that occurs or nil on success.
func LaunchInstances(c ConnectorI, name string, imageName string, flavorName string, count int, verbose bool) error {
	var err error
	if count <= 1 {
		// Take name as it is.
		err = c.LaunchServer(name, flavorName, imageName, verbose)
	} else {
		// Append index after the name of each instance.
		for i := 0; i < count; i++ {
			currErr := c.LaunchServer(fmt.Sprintf("%s-%d", name, (i+1)), flavorName, imageName, verbose)
			if err == nil {
				err = currErr
			}
		}
	}

	return err
}

/*
* Interface implementation
 */

type Connector struct {
	clientNova   *gophercloud.ServiceClient
	clientGlance *gophercloud.ServiceClient
}

// ListFlavors returns list of all flavors.
func (c *Connector) ListFlavors(minDiskGB int, minMemoryMB int) []flavors.Flavor {
	var flavs []flavors.Flavor = make([]flavors.Flavor, 0)

	pagerFlavors := flavors.ListDetail(c.clientNova, flavors.ListOpts{
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

// LaunchServer launches single server of given image.
func (c *Connector) LaunchServer(name string, flavorName string, imageName string, verbose bool) error {
	resp := servers.Create(c.clientNova, servers.CreateOpts{
		Name:          name,
		FlavorName:    flavorName,
		ImageName:     imageName,
		ServiceClient: c.clientNova, // need to pass this to perform name-to-ID lookup
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

// CreateImage creates image metadata on OpenStack.
func (c *Connector) CreateImage(name string, flavor *flavors.Flavor, verbose bool) (string, error) {
	createdImage, err := glanceImages.Create(c.clientGlance, glanceImages.CreateOpts{
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

// UploadImage uploads image binary data to existing OpenStack image metadata.
func (c *Connector) UploadImage(imageId string, filepath string, verbose bool) error {
	if verbose {
		fmt.Printf("Uploading composed image '%s' to OpenStack\n", filepath)
	}

	f, err := os.Open(filepath)
	if err != nil {
		fmt.Println(err)
	}
	defer f.Close()

	res := imagedata.Upload(c.clientGlance, imageId, f)
	return res.Err
}

// DeleteImage deletes image from OpenStack.
func (c *Connector) DeleteImage(imageId string) {
	glanceImages.Delete(c.clientGlance, imageId)
}

// GetFlavor returns flavors.Flavor struct for given flavorName.
func (c *Connector) GetFlavor(flavorName string, verbose bool) (*flavors.Flavor, error) {
	flavorId, err := flavors.IDFromName(c.clientNova, flavorName)
	if err != nil {
		return nil, err
	}

	flavor, err := flavors.Get(c.clientNova, flavorId).Extract()
	if err != nil {
		return nil, err
	}

	return flavor, nil
}

// GetImageMeta returns images.Image struct for given imageName.
func (c *Connector) GetImageMeta(imageName string, verbose bool) (*images.Image, error) {
	imageId, err := images.IDFromName(c.clientNova, imageName)
	if err != nil {
		return nil, err
	}

	image, err := images.Get(c.clientNova, imageId).Extract()
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
