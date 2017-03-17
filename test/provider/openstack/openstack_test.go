/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package openstack_test

import (
	"fmt"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/mikelangelo-project/capstan/provider/openstack"
	mocks "github.com/mikelangelo-project/capstan/test/mocks/provider/openstack"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type testingOpenstackSuite struct{}

var _ = Suite(&testingOpenstackSuite{})

type testPickFlavorManagerMock struct {
	mocks.OpenStackComputeManagerMockBase
	flavors []flavors.Flavor
}

func (m *testPickFlavorManagerMock) ListFlavors(minDiskGB int, minMemoryMB int) []flavors.Flavor {
	return m.flavors
}

func (s *testingOpenstackSuite) TestPickFlavor(c *C) {
	m := []struct {
		comment  string
		diskMB   int
		memoryMB int
		optimal  string
		flavors  []flavors.Flavor
		err      string
	}{
		{
			"disk size not specified",
			0, 0, "",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 1, RAM: 512, ID: "f01", Name: "name-f01"},
			},
			"Please specify disk size.",
		},
		{
			"no matching flavor",
			1 * 1024, 0, "",
			[]flavors.Flavor{},
			"No matching flavors to pick from",
		},
		{
			"take optimal disk size (edge)",
			20 * 1024, 0, "f03",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 40, RAM: 1024, ID: "f01"},
				flavors.Flavor{Disk: 30, RAM: 1024, ID: "f02"},
				flavors.Flavor{Disk: 20, RAM: 1024, ID: "f03"},
				flavors.Flavor{Disk: 25, RAM: 1024, ID: "f04"},
				flavors.Flavor{Disk: 35, RAM: 1024, ID: "f05"},
				flavors.Flavor{Disk: 45, RAM: 1024, ID: "f06"},
			},
			"",
		},
		{
			"take optimal disk size (non-edge)",
			15 * 1024, 0, "f03",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 40, RAM: 1024, ID: "f01"},
				flavors.Flavor{Disk: 30, RAM: 1024, ID: "f02"},
				flavors.Flavor{Disk: 20, RAM: 1024, ID: "f03"},
				flavors.Flavor{Disk: 25, RAM: 1024, ID: "f04"},
				flavors.Flavor{Disk: 35, RAM: 1024, ID: "f05"},
				flavors.Flavor{Disk: 45, RAM: 1024, ID: "f06"},
			},
			"",
		},
		{
			"take optimal disk size regardless memory",
			10 * 1024, 0, "f03",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 40, RAM: 1024, ID: "f01"},
				flavors.Flavor{Disk: 30, RAM: 1024, ID: "f02"},
				flavors.Flavor{Disk: 20, RAM: 4096, ID: "f03"},
				flavors.Flavor{Disk: 25, RAM: 1024, ID: "f04"},
				flavors.Flavor{Disk: 35, RAM: 1024, ID: "f05"},
				flavors.Flavor{Disk: 45, RAM: 1024, ID: "f06"},
			},
			"",
		},
		{
			"take optimal memory when equal disk size",
			10 * 1024, 0, "f01",
			[]flavors.Flavor{
				flavors.Flavor{Disk: 10, RAM: 1024, ID: "f01"},
				flavors.Flavor{Disk: 10, RAM: 8192, ID: "f02"},
				flavors.Flavor{Disk: 10, RAM: 4096, ID: "f03"},
				flavors.Flavor{Disk: 10, RAM: 2048, ID: "f04"},
			},
			"",
		},
	}
	for _, args := range m {
		c.Logf("CASE: %s", args.comment)

		manager := &testPickFlavorManagerMock{flavors: args.flavors}

		// This is what we're testing here.
		flavor, err := openstack.PickFlavor(manager, int64(args.diskMB), int64(args.memoryMB), false)

		// Expectations.
		if args.err != "" {
			c.Check(err, ErrorMatches, args.err)
		} else {
			c.Check(flavor, NotNil)
			c.Check(err, IsNil)
			c.Check(flavor.ID, Equals, args.optimal)
		}
	}
}

func (s *testingOpenstackSuite) TestPickFlavorWithErrors(c *C) {
	manager := &mocks.OpenStackComputeManagerMockBase{}

	// This is what we're testing here.
	flavor, err := openstack.PickFlavor(manager, 10*1024, 2048, false)

	// Expectations.
	c.Check(flavor, IsNil)
	c.Check(err, NotNil)
	c.Check(err, ErrorMatches, "No matching flavors to pick from")
}

type testGetOrPickFlavorManagerMock struct {
	mocks.OpenStackComputeManagerMockBase
}

func (m *testGetOrPickFlavorManagerMock) ListFlavors(minDiskGB int, minMemoryMB int) []flavors.Flavor {
	return []flavors.Flavor{
		flavors.Flavor{ID: "flavor-01", Name: "foo"},
	}
}

func (m *testGetOrPickFlavorManagerMock) GetFlavor(flavorName string, verbose bool) (*flavors.Flavor, error) {
	return &flavors.Flavor{Name: flavorName}, nil
}

func (s *testingOpenstackSuite) TestGetOrPickFlavor(c *C) {
	m := []struct {
		comment    string
		flavorName string
		diskMB     int
		memoryMB   int
	}{
		{
			"get flavor by name",
			"flavor1",
			0, 0,
		},
		{
			"get flavor by criteria",
			"",
			10 * 1024, 2048,
		},
		{
			"criteria is ignored when flavor name present",
			"flavor1",
			10 * 1024, 2048,
		},
	}
	for _, args := range m {
		c.Logf("CASE: %s", args.comment)
		manager := &testGetOrPickFlavorManagerMock{}

		// This is what we're testing here.
		flavor, err := openstack.GetOrPickFlavor(manager, args.flavorName, int64(args.diskMB), int64(args.memoryMB), false)

		// Expectations.
		c.Assert(flavor, NotNil)
		c.Check(err, IsNil)
		if args.flavorName != "" {
			c.Check(flavor.Name, Equals, args.flavorName)
		} else {
			c.Check(flavor.ID, Equals, "flavor-01")
		}

	}
}

func (s *testingOpenstackSuite) TestGetOrPickFlavorWithErrors(c *C) {
	m := []struct {
		comment    string
		flavorName string
		err        string
	}{
		{
			"get flavor by name",
			"flavor1",
			"mock not implemented",
		},
		{
			"get flavor by criteria",
			"",
			"No matching flavors to pick from",
		},
	}
	for _, args := range m {
		manager := &mocks.OpenStackComputeManagerMockBase{}

		// This is what we're testing here.
		_, err := openstack.GetOrPickFlavor(manager, args.flavorName, 10*1024, 2048, false)

		// Expectations.
		c.Check(err, NotNil)
		c.Check(err, ErrorMatches, args.err)
	}
}

type testPushImageManagerMock struct {
	mocks.OpenStackImageManagerMockBase
	CreateImage_Name     string
	CreateImage_Flavor   string
	UploadImage_Filepath string
}

func (m *testPushImageManagerMock) CreateImage(name string, flavor *flavors.Flavor, verbose bool) (string, error) {
	m.CreateImage_Name = name
	m.CreateImage_Flavor = flavor.ID
	return "created-image-id", nil
}

func (m *testPushImageManagerMock) UploadImage(imageId string, filepath string, verbose bool) error {
	m.UploadImage_Filepath = filepath
	return fmt.Errorf("mock not implemented")
}

func (s *testingOpenstackSuite) TestPushImage(c *C) {
	m := []struct {
		comment   string
		imageName string
		imagePath string
		flavor    flavors.Flavor
	}{
		{
			"push image creates image meta and uploads data",
			"img01", "/home/dir/file.qemu",
			flavors.Flavor{ID: "flavor-01"},
		},
	}
	for _, args := range m {
		c.Logf("CASE: %s", args.comment)
		manager := &testPushImageManagerMock{}

		// This is what we're testing here.
		openstack.PushImage(manager, args.imageName, args.imagePath, &args.flavor, false)

		// Expectations.
		c.Check(manager.CreateImage_Name, Equals, args.imageName)
		c.Check(manager.CreateImage_Flavor, Equals, args.flavor.ID)
		c.Check(manager.UploadImage_Filepath, Equals, args.imagePath)
	}
}

func (s *testingOpenstackSuite) TestPushImageWithErrors(c *C) {
	manager := &mocks.OpenStackImageManagerMockBase{}

	// This is what we're testing here.
	openstack.PushImage(manager, "image-name", "/image/path", &flavors.Flavor{ID: "flavor-01"}, false)

	// Expectations.
	// TODO: refactor PushImage so that it returns error - it should fail when manager fails!
}
