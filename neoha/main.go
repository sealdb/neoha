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

package main

import (
	nlog "neoha/base/nlog"
	build "neoha/build"
	config "neoha/config"
	//build "github.com/sealdb/neoha/build"
	//config "github.com/sealdb/neoha/config"
	//nlog "github.com/sealdb/neoha/base/nlog"
	"flag"
	"fmt"
	"os"
	//"raft"
	//"server"
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
	log.Warning("etcd:[%+v]", conf.Etcd)

	log.Warning("raft:[%+v]", conf.Raft)
	log.Warning("raft.partner_address:[%+v]\n", conf.Raft.PartnerAddrs)

	log.Warning("bootstrap:")
	log.Warning("bootstrap.dcs:[%+v]", conf.Bootstrap.DcsConf)
	log.Warning("bootstrap.dcs.standby_cluster:[%+v]", conf.Bootstrap.DcsConf.StandbyCluster)
	log.Warning("bootstrap.dcs.postgresql:[%+v]", conf.Bootstrap.DcsConf.DcsPostgresql)
	log.Warning("bootstrap.dcs.initdb:[%+v]", conf.Bootstrap.InitDB)
	log.Warning("bootstrap.dcs.pg_hba:[%+v]", conf.Bootstrap.PgHba)
	log.Warning("bootstrap.dcs.post_init:[%+v]", conf.Bootstrap.PostInit)
	log.Warning("bootstrap.dcs.users:[%+v]\n", conf.Bootstrap.Users)

	log.Warning("postgresql:[%+v]", conf.Postgresql)
	log.Warning("postgresql.authentication.replication:[%+v]", conf.Postgresql.Auth.Repl)
	log.Warning("postgresql.authentication.superuser:[%+v]", conf.Postgresql.Auth.SuperUser)
	log.Warning("postgresql.authentication.rewind:[%+v]\n", conf.Postgresql.Auth.Rewind)

	log.Warning("watchdog:[%+v]", conf.Watchdog)
	log.Warning("tags:[%+v]", conf.Tags)
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

	// set the initialization state
	//switch flag_role {
	//case "LEADER":
	//	state = raft.LEADER
	//case "FOLLOWER":
	//	state = raft.FOLLOWER
	//case "IDLE":
	//	state = raft.IDLE
	//default:
	//	state = raft.UNKNOW
	//}

	// build
	log.Info("main: tag=[%s], git=[%s], goversion=[%s], builddate=[%s]",
		build.Tag, build.Git, build.GoVersion, build.Time)

	// server
	//server := server.NewServer(conf, log, state)
	//server.Init()
	//server.Start()
	//log.Info("neoha.start.success...")
	//
	//if conf.Server.EnableAPIs {
	//	// Admin portal.
	//	admin := ctl.NewAdmin(log, server)
	//	admin.Start()
	//	defer admin.Stop()
	//}
	//
	//server.Wait()

	log.Info("neoha shutdown complete")
}
