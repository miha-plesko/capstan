/*
 * Copyright (C) 2017 XLAB, Ltd.
 *
 * This work is open source software, licensed under the terms of the
 * BSD license as described in the LICENSE file in the top-level directory.
 */

package openstack_test

import (
	"testing"

	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/mikelangelo-project/capstan/provider/openstack"
	"github.com/mikelangelo-project/capstan/test/mocks"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type testingOpenstackSuite struct{}

var _ = Suite(&testingOpenstackSuite{})

func usualFlavors() []flavors.Flavor {
	return []flavors.Flavor{
		flavors.Flavor{Disk: 1, RAM: 512, ID: "f01", Name: "name-f01"},
		flavors.Flavor{Disk: 1, RAM: 1024, ID: "f02", Name: "name-f02"},
		flavors.Flavor{Disk: 10, RAM: 512, ID: "f03", Name: "name-f03"},
		flavors.Flavor{Disk: 10, RAM: 2048, ID: "f04", Name: "name-f04"},
		flavors.Flavor{Disk: 10, RAM: 1024, ID: "f05", Name: "name-f05"},
		flavors.Flavor{Disk: 40, RAM: 1024, ID: "f06", Name: "name-f06"},
		flavors.Flavor{Disk: 20, RAM: 1024, ID: "f07", Name: "name-f07"},
		flavors.Flavor{Disk: 20, RAM: 4096, ID: "f08", Name: "name-f08"},
		flavors.Flavor{Disk: 20, RAM: 2048, ID: "f09", Name: "name-f09"},
		flavors.Flavor{Disk: 20, RAM: 512, ID: "f10", Name: "name-f10"},
		flavors.Flavor{Disk: 100, RAM: 4096, ID: "f11", Name: "name-f11"},
		flavors.Flavor{Disk: 100, RAM: 9999, ID: "f12", Name: "name-f12"},
		flavors.Flavor{Disk: 100, RAM: 1, ID: "f13", Name: "name-f13"},
	}
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
			[]flavors.Flavor{},
			"Please specify disk size.",
		},
		{
			"no matching flavor",
			1 * 1024, 0, "",
			[]flavors.Flavor{},
			"No matching flavors to pick from",
		},
		{
			"take optimal disk size",
			40 * 1024, 0, "f06",
			usualFlavors(),
			"",
		},
		{
			"prefer smaller disk for equal memory",
			1 * 1024, 1024, "f02",
			usualFlavors(),
			"",
		},
		{
			"prefer smaller disk regardless memory",
			20 * 1024, 0, "f10",
			usualFlavors(),
			"",
		},
	}
	for _, args := range m {
		c.Logf("CASE: %s", args.comment)
		connector := &mocks.ConnectorMock{Flavors: args.flavors}

		// This is what we're testing here.
		flavor, err := openstack.PickFlavor(connector, int64(args.diskMB), int64(args.memoryMB), false)

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

func (s *testingOpenstackSuite) TestGetOrPickFlavor(c *C) {
	m := []struct {
		comment    string
		flavorName string
		audit      []string
	}{
		{
			"get flavor by name",
			"flavor1",
			[]string{"GetFlavor(flavor1, false)"},
		},
		{
			"get flavor by criteria",
			"",
			[]string{"ListFlavors(1, 0)"},
		},
	}
	for _, args := range m {
		c.Logf("CASE: %s", args.comment)
		connector := &mocks.ConnectorMock{Flavors: usualFlavors()}

		// This is what we're testing here.
		openstack.GetOrPickFlavor(connector, args.flavorName, 1, 0, false)

		// Expectations.
		c.Check(connector.GetAudit(), DeepEquals, args.audit)
	}
}

func (s *testingOpenstackSuite) TestPushImage(c *C) {
	m := []struct {
		comment   string
		imageName string
		imagePath string
		audit     []string
	}{
		{
			"push image creates image meta and uploads data",
			"img01", "/home/dir/file.qemu",
			[]string{
				"CreateImage(img01, f01, false)",
				"UploadImage(id-img01, /home/dir/file.qemu, false)",
			},
		},
	}
	for _, args := range m {
		c.Logf("CASE: %s", args.comment)
		connector := &mocks.ConnectorMock{Flavors: usualFlavors()}

		// This is what we're testing here.
		openstack.PushImage(connector, args.imageName, args.imagePath, &connector.Flavors[0], false)

		// Expectations.
		c.Check(connector.GetAudit(), DeepEquals, args.audit)
	}
}
