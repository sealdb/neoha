/*
 * Copyright 2022-2025 The NeoHA Authors.
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

package main

import (
	"flag"
	"fmt"
	"os"

	"neoha/base/nlog"
	"neoha/build"
	"neoha/config"
	"neoha/server"
)

var (
	flag_conf string
	flag_role string
)

func init() {
	flag.StringVar(&flag_conf, "c", "", "neoha config file")
	flag.StringVar(&flag_conf, "config", "", "neoha config file")
	flag.StringVar(&flag_role, "r", "", "role type:[LEADER|FOLLOWER|IDLE]")
	flag.StringVar(&flag_role, "role", "", "role type:[LEADER|FOLLOWER|IDLE]")
}

func printConfig(conf *config.Config, log *nlog.Log) {
	log.Warning("neoha.conf:")
	log.Warning("scope:[%+v]", conf.Scope)
	log.Warning("namespace:[%+v]", conf.NameSpace)
	log.Warning("name:[%+v]\n", conf.Name)

	log.Warning("restapi:[%+v]", conf.RestAPI)
	log.Warning("restapi.authentication:[%+v]\n", conf.RestAPI.Authentication)

	log.Warning("ctl:[%+v]", conf.Ctl)

	log.Warning("election:")
	log.Warning("etcd:[%+v]", conf.Election.Etcd)
	log.Warning("raft:[%+v]", conf.Election.Raft)

	log.Warning("bootstrap:")
	log.Warning("bootstrap.postgresql:[%+v]", conf.Bootstrap.BootstrapPostgresql)
	log.Warning("bootstrap.mysql:[%+v]", conf.Bootstrap.BootstrapMysql)

	log.Warning("database:")
	log.Warning("database.postgresql:[%+v]", conf.Database.Postgresql)
	log.Warning("database.mysql:[%+v]", conf.Database.Mysql)

	log.Warning("watchdog:[%+v]", conf.Watchdog)
	log.Warning("tags:[%+v]", conf.Tags)
}

func getServerState(role *string) server.ServerState {
	switch flag_role {
	case "LEADER":
		return server.ServerLeader
	case "FOLLOWER":
		return server.ServerFollower
	case "IDLE":
		return server.ServerIdle
	case "":
		return server.ServerLearner
	default:
		return server.ServerUnknown
	}
}

func main() {
	log := nlog.NewStdLog(nlog.Level(nlog.DEBUG))
	flag.Parse()

	build := build.GetInfo()
	fmt.Printf("neoha:[%+v]\n", build)
	if flag_conf == "" {
		fmt.Printf("usage: %s [-c|--config <neoha_config_file>]\n", os.Args[0])
		os.Exit(1)
	}

	// config
	conf, err := config.LoadConfig(flag_conf)
	if err != nil {
		log.Panic("neoha.loadconfig.error[%v]", err)
	}
	printConfig(conf, log)

	// set log level
	log.SetLevel(conf.Log.Level)

	// build
	log.Info("main: tag=[%s], git=[%s], goversion=[%s], builddate=[%s]",
		build.Tag, build.Git, build.GoVersion, build.Time)

	// server
	server := server.NewServer(conf, log, getServerState(&flag_role))
	server.Init()
	server.Start()

	//if conf.Server.EnableAPIs {
	//	// Admin portal.
	//	admin := ctl.NewAdmin(log, server)
	//	admin.Start()
	//	defer admin.Stop()
	//}

	log.Info("neoha.start.success...")

	server.Wait()

	log.Info("neoha shutdown complete")
}
