package main

import (
	"fmt"
	"os"
	"testing"

	"github.com/gophercloud/gophercloud/openstack/sharedfilesystems/v2/shares"
	"github.com/pepabo/go-netapp/netapp"

	_ "github.com/motemen/go-loghttp/global"
	"github.com/stretchr/testify/assert"
)

func TestNetappVserver(t *testing.T) {
	host := os.Getenv("NETAPP_HOST")
	username := os.Getenv("NETAPP_USERNAME")
	password := os.Getenv("NETAPP_PASSWORD")
	c := newNetappClient(host, username, password)

	v := c.VServer

	p := &netapp.VServerOptions{
		DesiredAttributes: &netapp.VServerQuery{
			VServerInfo: &netapp.VServerInfo{
				VserverName:   "1",
				UUID:          "1",
				State:         "1",
				AggregateList: &[]string{"x"},
			},
		},
		MaxRecords: 100,
	}

	r, _, _ := v.List(p)
	// r.Results.NumRecords <= 100

	assert.True(t, r.Results.Passed())
	assert.NotNil(t, r.Results.AttributesList.VserverInfo[0])
}

func TestNetappVolume(t *testing.T) {
	host := os.Getenv("NETAPP_HOST")
	username := os.Getenv("NETAPP_USERNAME")
	password := os.Getenv("NETAPP_PASSWORD")
	c := newNetappClient(host, username, password)

	v := c.VolumeSpace
	p := netapp.VolumeSpaceOptions{
		MaxRecords: 10,
	}

	r, _, _ := v.List(&p)

	assert.True(t, r.Results.Passed())
	assert.Equal(t, "", r.Results)
	// assert.NotNil(t, r.Results)
}

func TestNetappVolumePage(t *testing.T) {
	host := os.Getenv("NETAPP_HOST")
	username := os.Getenv("NETAPP_USERNAME")
	password := os.Getenv("NETAPP_PASSWORD")
	c := NewFiler("testFiler", host, username, password)

	opts := netapp.VolumeOptions{
		MaxRecords: 10,
	}

	volumePages := c.getNetappVolumePages(&opts, 1)

	vols := extracVolumes(volumePages)
	if assert.NotNil(t, vols) {
		vol := vols[0]
		fmt.Println("# of Vols: ", len(vols))

		fmt.Println("\nFirst volume:")
		fmt.Println("Vserver\t\t", vol.VolumeIDAttributes.OwningVserverName)
		fmt.Println("Name\t\t", vol.VolumeIDAttributes.Name)
		fmt.Println("Type\t\t", vol.VolumeIDAttributes.Type)
		fmt.Println("Node\t\t", vol.VolumeIDAttributes.Node)
		fmt.Println("AvailableSize\t", vol.VolumeSpaceAttributes.SizeAvailable)
		fmt.Println("Percentage\t", vol.VolumeSpaceAttributes.PercentageSizeUsed)

	}
}

func TestManilaClient(t *testing.T) {
	var err error

	client, err := newManilaClient()
	if assert.Nil(t, err) {

		lo := shares.ListOpts{AllTenants: true}
		allpages, err := shares.ListDetail(client, lo).AllPages()
		if assert.Nil(t, err) {

			sh, _ := shares.ExtractShares(allpages)
			s := sh[0]
			fmt.Println("ID\t\t", s.ID)
			fmt.Println("Name\t\t", s.Name)
			fmt.Println("ProjectID\t", s.ProjectID)
			fmt.Println("ShareServerID\t", s.ShareServerID)
			fmt.Println("ShareType\t", s.ShareType)
			fmt.Println("AvailabilityZone", s.AvailabilityZone)
			// fmt.Printf("%T %+v", s, s)

			sel, err := shares.GetExportLocations(client, s.ID).Extract()
			if assert.Nil(t, err) {
				if assert.NotNil(t, sel[0].ShareInstanceID) {
					fmt.Println("ShareInstanceID\t", sel[0].ShareInstanceID)
				}
			}
		}
	}

	if err != nil {
		assert.Equal(t, "", err.Error())
	}
}
