/*
 * Copyright 2022 The NeoHA Authors.
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
*/

package neohactl

import (
	"fmt"
	"neoha/base/common"
	"strings"

	"github.com/spf13/cobra"
)

var (
	addrStr     string
	vipStr      string
	replUserStr string
	replPwdStr  string
	portInt     int
)

func NewInitCommand() *cobra.Command { // TODO: need user custom
	cmd := &cobra.Command{
		Use:   "init",
		Short: "init the neoha config file",
		Long: `init the neoha config file.

			steps:
			1.set endpoint
			2.set vip command
			3.set repl user and pwd
			`,
		Run: initCommandFn,
	}

	cmd.Flags().StringVar(&addrStr, "address", "", "--address=<ip>")
	cmd.Flags().IntVar(&portInt, "port", 0, "--port=<port>")
	cmd.Flags().StringVar(&vipStr, "vip", "", "--vip=<vip>")
	cmd.Flags().StringVar(&replUserStr, "repluser", "", "--repluser=<repluser>")
	cmd.Flags().StringVar(&replPwdStr, "replpwd", "", "--replpwd=<replpwd>")

	return cmd
}

// initCommandFn
func initCommandFn(cmd *cobra.Command, args []string) {
	if !checkInitFlags() {
		cmd.Usage()
		return
	}

	//eth, err := getEth(addrStr)
	//ErrorOK(err)

	conf, err := GetConfig()
	ErrorOK(err)
	//// server
	//conf.Server.Endpoint = fmt.Sprintf("%v:%v", addrStr, portInt)
	//
	//// replication
	//conf.Replication.User = replUserStr
	//conf.Replication.Passwd = replPwdStr
	//
	//// server
	//conf.Raft.LeaderStartCommand = fmt.Sprintf("ip a d %s/32 dev %s", vipStr, eth)
	//conf.Raft.LeaderStopCommand = fmt.Sprintf("ip a a %s/32 dev %s", vipStr, eth)
	//
	//// backup
	//conf.Backup.SSHHost = addrStr

	err = SaveConfig(conf)
	ErrorOK(err)
}

func checkInitFlags() bool {
	if addrStr == "" ||
		portInt == 0 ||
		vipStr == "" ||
		replUserStr == "" ||
		replPwdStr == "" {
		log.Error("cmd.init.flags.address[%v].port[%v].vip[%v]",
			addrStr, portInt, vipStr)
		return false
	}

	return true
}

// get the eth via ip address
func getEth(ip string) (string, error) {
	bash := "bash"
	args := []string{
		"-c",
		fmt.Sprintf("ifconfig | grep -B 1 'inet addr:%s' | grep HWaddr | awk '{print $1}'", ip),
	}
	result, err := common.RunCommand(bash, args...)
	if err != nil {
		msg := fmt.Sprintf("cmd.init.getEth[%v].error[%v]", ip, err)
		log.Error(msg)
		return "", fmt.Errorf(msg)
	}

	ret := strings.TrimSpace(result)
	if ret == "" {
		return "", fmt.Errorf("getEth[%v].can.not.found.eth", ip)
	}

	return ret, nil
}
