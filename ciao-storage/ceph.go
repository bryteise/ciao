// Copyright (c) 2016 Intel Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package storage

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"

	"github.com/01org/ciao/ssntp/uuid"
)

// CephDriver maintains context for the ceph driver interface.
type CephDriver struct {
	// ID is the cephx user ID to use
	ID string
}

// CreateBlockDevice will create a rbd image in the ceph cluster.
func (d CephDriver) CreateBlockDevice(volumeUUID string, imagePath string, size int) (BlockDevice, error) {
	if volumeUUID == "" {
		volumeUUID = uuid.Generate().String()
	} else {
		_, err := uuid.Parse(volumeUUID)
		if err != nil {
			return BlockDevice{}, fmt.Errorf("invalid UUID supplied for volume ID")
		}
	}

	var cmd *exec.Cmd

	// imageFeatures holds the image features to use when creating a ceph rbd image format 2
	// Currently the kernel rdb client only supports layering but in the future more feaures
	// should be added as they are enabled in the kernel.
	if imagePath != "" {
		rbdStr := fmt.Sprintf("rbd:rbd/%s:id=%s", volumeUUID, d.ID)
		cmd = exec.Command("qemu-img", "convert", "-O", "rbd", imagePath, rbdStr)
	} else {
		// create an empty volume
		cmd = exec.Command("rbd", "--id", d.ID, "--image-feature", "layering", "create", "--size", strconv.Itoa(size)+"G", volumeUUID)
	}

	err := cmd.Run()
	if err != nil {
		return BlockDevice{}, fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}

	return BlockDevice{ID: volumeUUID}, nil
}

// CreateBlockDeviceFromSnapshot will create a block device derived from the previously created snapshot.
func (d CephDriver) CreateBlockDeviceFromSnapshot(volumeUUID string, snapshotID string) (BlockDevice, error) {
	ID := uuid.Generate().String()

	var cmd *exec.Cmd

	cmd = exec.Command("rbd", "--id", d.ID, "clone", volumeUUID+"@"+snapshotID, ID)

	err := cmd.Run()
	if err != nil {
		return BlockDevice{}, fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}

	return BlockDevice{ID: ID}, nil
}

// CreateBlockDeviceSnapshot creates and protects the snapshot with the provided name
func (d CephDriver) CreateBlockDeviceSnapshot(volumeUUID string, snapshotID string) error {
	var cmd *exec.Cmd
	cmd = exec.Command("rbd", "--id", d.ID, "snap", "create", volumeUUID+"@"+snapshotID)

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}

	cmd = exec.Command("rbd", "--id", d.ID, "snap", "protect", volumeUUID+"@"+snapshotID)

	err = cmd.Run()
	if err != nil {
		d.DeleteBlockDevice(volumeUUID)
		return fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}
	return nil
}

// CopyBlockDevice will copy an existing volume
func (d CephDriver) CopyBlockDevice(volumeUUID string) (BlockDevice, error) {
	ID := uuid.Generate().String()

	var cmd *exec.Cmd

	cmd = exec.Command("rbd", "--id", d.ID, "cp", volumeUUID, ID)

	err := cmd.Run()
	if err != nil {
		return BlockDevice{}, fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}

	return BlockDevice{ID: ID}, nil
}

// DeleteBlockDevice will remove a rbd image from the ceph cluster.
func (d CephDriver) DeleteBlockDevice(volumeUUID string) error {
	cmd := exec.Command("rbd", "--id", d.ID, "rm", volumeUUID)
	return cmd.Run()
}

// DeleteBlockDeviceSnapshot unprotects and deletes the snapshot with the provided name
func (d CephDriver) DeleteBlockDeviceSnapshot(volumeUUID string, snapshotID string) error {
	var cmd *exec.Cmd

	cmd = exec.Command("rbd", "--id", d.ID, "snap", "unprotect", volumeUUID+"@"+snapshotID)
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}

	cmd = exec.Command("rbd", "--id", d.ID, "snap", "rm", volumeUUID+"@"+snapshotID)
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("Error when running: %v: %v", cmd.Args, err)
	}
	return nil
}

func (d CephDriver) getCredentials() []string {
	args := make([]string, 0, 8)
	if d.ID != "" {
		args = append(args, "--id", d.ID)
	}
	return args
}

// MapVolumeToNode maps a ceph volume to a rbd device on a node.  The
// path to the new device is returned if the mapping succeeds.
func (d CephDriver) MapVolumeToNode(volumeUUID string) (string, error) {
	args := append(d.getCredentials(), "map", volumeUUID)
	cmd := exec.Command("rbd", args...)
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewBuffer(data))
	if !scanner.Scan() {
		return "", fmt.Errorf("Unable to determine device name for %s", volumeUUID)
	}
	return scanner.Text(), nil
}

// UnmapVolumeFromNode unmaps a ceph volume from a local device on a node.
func (d CephDriver) UnmapVolumeFromNode(volumeUUID string) error {
	args := append(d.getCredentials(), "unmap", volumeUUID)
	return exec.Command("rbd", args...).Run()
}

// GetVolumeMapping returns a map of volumeUUID to mapped devices.
func (d CephDriver) GetVolumeMapping() (map[string][]string, error) {
	args := append(d.getCredentials(), "showmapped", "--format", "json")
	cmd := exec.Command("rbd", args...)
	data, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	vmap := map[string]struct {
		Name   string `json:"name"`
		Device string `json:"device"`
	}{}
	err = json.Unmarshal([]byte(data), &vmap)
	if err != nil {
		return nil, fmt.Errorf("Unable to parse output from rbd show mapped: %v", err)
	}

	volumeDevMap := make(map[string][]string)

	for _, v := range vmap {
		volumeDevMap[v.Name] = append(volumeDevMap[v.Name], v.Device)
	}

	return volumeDevMap, nil
}
