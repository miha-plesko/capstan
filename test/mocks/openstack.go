/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package mocks

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/images"
	"github.com/mikelangelo-project/capstan/provider/openstack"
)

type ConnectorMock struct {
	Flavors []flavors.Flavor
	audit   []string // internal
}

var _ openstack.ConnectorI = (*ConnectorMock)(nil)

// ListFlavors mock returns those flavors that match criteria.
func (c *ConnectorMock) ListFlavors(minDiskGB int, minMemoryMB int) []flavors.Flavor {
	c.addAudit("ListFlavors", minDiskGB, minMemoryMB)
	var matchingFlavors []flavors.Flavor = make([]flavors.Flavor, 0)
	for _, f := range c.Flavors {
		curr := f
		if f.Disk >= minDiskGB && f.RAM >= minMemoryMB {
			matchingFlavors = append(matchingFlavors, curr)
		}
	}
	return matchingFlavors
}

func (c *ConnectorMock) LaunchServer(name string, flavorName string, imageName string, verbose bool) error {
	c.addAudit("LaunchServer", name, flavorName, imageName, verbose)
	return nil
}

// CreateImage mock always returns image id of form "id-{NAME}".
func (c *ConnectorMock) CreateImage(name string, flavor *flavors.Flavor, verbose bool) (string, error) {
	c.addAudit("CreateImage", name, flavor, verbose)
	return "id-" + name, nil
}

func (c *ConnectorMock) UploadImage(imageId string, filepath string, verbose bool) error {
	c.addAudit("UploadImage", imageId, filepath, verbose)
	return nil
}

func (c *ConnectorMock) DeleteImage(imageId string) {
	c.addAudit("DeleteImage", imageId)
}

func (c *ConnectorMock) GetFlavor(flavorName string, verbose bool) (*flavors.Flavor, error) {
	c.addAudit("GetFlavor", flavorName, verbose)

	for _, f := range c.Flavors {
		if f.Name == flavorName {
			return &f, nil
		}
	}
	return nil, fmt.Errorf("mock: flavor not found")
}

func (c *ConnectorMock) GetImageMeta(imageName string, verbose bool) (*images.Image, error) {
	c.addAudit("GetImageMeta", imageName, verbose)
	// TODO
	return nil, nil
}

/*
* Utility function for detailed unit tests
 */

func (c *ConnectorMock) addAudit(msg string, args ...interface{}) {
	if c.audit == nil {
		c.audit = make([]string, 0)
	}
	msg += "("
	for _, arg := range args {
		if s, ok := arg.(int); ok {
			msg += strconv.FormatInt(int64(s), 10)
		} else if s, ok := arg.(int64); ok {
			msg += strconv.FormatInt(s, 10)
		} else if s, ok := arg.(string); ok {
			msg += s
		} else if s, ok := arg.(bool); ok {
			msg += strconv.FormatBool(s)
		} else if s, ok := arg.(*flavors.Flavor); ok {
			msg += s.ID
		} else {
			msg += fmt.Sprintf("%b", arg)
		}
		msg += ", "
	}
	msg = strings.TrimSuffix(msg, ", ")
	msg += ")"
	c.audit = append(c.audit, msg)
}

func (c *ConnectorMock) GetAudit() []string {
	return c.audit
}
