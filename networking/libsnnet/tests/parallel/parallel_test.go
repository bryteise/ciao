//
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
//

package parallel

import (
	"net"
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/01org/ciao/networking/libsnnet"
)

var cnNetEnv string

var scaleCfg = struct {
	maxBridgesShort int
	maxVnicsShort   int
	maxBridgesLong  int
	maxVnicsLong    int
}{2, 64, 64, 64}

func cninit() {
	cnNetEnv = os.Getenv("SNNET_ENV")

	if cnNetEnv == "" {
		cnNetEnv = "10.3.66.0/24"
	}
}

func TestCN_Parallel(t *testing.T) {

	cn := &libsnnet.ComputeNode{}

	cn.NetworkConfig = &libsnnet.NetworkConfig{
		ManagementNet: nil,
		ComputeNet:    nil,
		Mode:          libsnnet.GreTunnel,
	}

	cn.ID = "cnuuid"

	cninit()
	_, mnet, _ := net.ParseCIDR(cnNetEnv)

	//From YAML, on agent init
	mgtNet := []net.IPNet{*mnet}
	cn.ManagementNet = mgtNet
	cn.ComputeNet = mgtNet

	if err := cn.Init(); err != nil {
		t.Fatal("ERROR: cn.Init failed", err)
	}
	if err := cn.ResetNetwork(); err != nil {
		t.Error("ERROR: cn.ResetNetwork failed", err)
	}
	if err := cn.DbRebuild(nil); err != nil {
		t.Fatal("ERROR: cn.dbRebuild failed")
	}

	//From YAML on instance init
	tenantID := "tenantuuid"
	concIP := net.IPv4(192, 168, 254, 1)

	var maxBridges, maxVnics int
	if testing.Short() {
		maxBridges = scaleCfg.maxBridgesShort
		maxVnics = scaleCfg.maxVnicsShort
	} else {
		maxBridges = scaleCfg.maxBridgesLong
		maxVnics = scaleCfg.maxVnicsLong
	}

	channelSize := maxBridges*maxVnics + 1
	createCh := make(chan *libsnnet.VnicConfig, channelSize)
	destroyCh := make(chan *libsnnet.VnicConfig, channelSize)

	for s3 := 1; s3 <= maxBridges; s3++ {
		s4 := 0
		_, tenantNet, _ := net.ParseCIDR("192.168." + strconv.Itoa(s3) + "." + strconv.Itoa(s4) + "/24")
		subnetID := "suuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)

		for s4 := 2; s4 <= maxVnics; s4++ {

			vnicIP := net.IPv4(192, 168, byte(s3), byte(s4))
			mac, _ := net.ParseMAC("CA:FE:00:01:02:03")

			vnicID := "vuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			instanceID := "iuuid_" + strconv.Itoa(s3) + "_" + strconv.Itoa(s4)
			vnicCfg := &libsnnet.VnicConfig{
				VnicIP:     vnicIP,
				ConcIP:     concIP,
				VnicMAC:    mac,
				Subnet:     *tenantNet,
				SubnetKey:  s3,
				VnicID:     vnicID,
				InstanceID: instanceID,
				SubnetID:   subnetID,
				TenantID:   tenantID,
				ConcID:     "cnciuuid",
			}

			createCh <- vnicCfg
			destroyCh <- vnicCfg
		}
	}

	close(createCh)
	close(destroyCh)

	var wg sync.WaitGroup
	wg.Add(len(createCh))

	for vnicCfg := range createCh {
		go func(vnicCfg *libsnnet.VnicConfig) {
			defer wg.Done()
			if vnicCfg == nil {
				t.Errorf("WARNING: VNIC nil")
				return
			}
			if _, _, _, err := cn.CreateVnicV2(vnicCfg); err != nil {
				t.Fatal("ERROR: cn.CreateVnicV2  failed", err)
			}
		}(vnicCfg)
	}

	wg.Wait()

	wg.Add(len(destroyCh))

	for vnicCfg := range destroyCh {
		go func(vnicCfg *libsnnet.VnicConfig) {
			defer wg.Done()
			if vnicCfg == nil {
				t.Errorf("WARNING: VNIC nil")
				return
			}
			if _, _, err := cn.DestroyVnicV2(vnicCfg); err != nil {
				t.Fatal("ERROR: cn.DestroyVnicV2 failed event", vnicCfg, err)
			}
		}(vnicCfg)
	}

	wg.Wait()
}