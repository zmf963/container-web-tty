package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"gopkg.in/urfave/cli.v2"

	"github.com/wrfly/container-web-tty/config"
)

func main() {
	conf := config.New()
	appFlags := []cli.Flag{
		&cli.StringFlag{
			Name:        "port",
			Aliases:     []string{"p"},
			EnvVars:     envVars("port"),
			Usage:       "server port",
			Value:       "8080",
			Destination: &conf.Server.Port,
		},
		&cli.BoolFlag{
			Name:        "debug",
			Aliases:     []string{"d"},
			Value:       false,
			EnvVars:     envVars("debug"),
			Usage:       "debug mode",
			Destination: &conf.Debug,
		},
		&cli.StringFlag{
			Name:        "backend",
			Aliases:     []string{"b"},
			EnvVars:     envVars("backend"),
			Value:       "docker",
			Usage:       "backend type, 'docker' or 'kube' for now",
			Destination: &conf.Backend.Type,
		},
		&cli.StringFlag{
			Name:        "docker-host",
			EnvVars:     append(envVars("docker-host"), "DOCKER_HOST"),
			Value:       "/var/run/docker.sock",
			Usage:       "docker host path",
			Destination: &conf.Backend.Docker.DockerHost,
		},
		&cli.StringFlag{
			Name:        "docker-ps",
			EnvVars:     envVars("docker-ps"),
			Usage:       "docker ps options",
			Destination: &conf.Backend.Docker.PsOptions,
		},
		&cli.StringFlag{
			Name:        "kube-config",
			EnvVars:     envVars("kube-config"),
			Value:       kubeConfigPath(),
			Usage:       "kube config path",
			Destination: &conf.Backend.Kube.ConfigPath,
		},
		&cli.StringFlag{
			Name:    "extra-args",
			EnvVars: envVars("extra-args"),
			Usage:   "pass extra args to the backend",
		},
		&cli.StringFlag{
			Name:    "servers",
			EnvVars: envVars("servers"),
			Usage:   "upstream servers, for proxy mode",
		},
		&cli.StringFlag{
			Name:    "idle-time",
			EnvVars: envVars("idle-time"),
			Usage:   "time out of an idle connection",
		},
		&cli.BoolFlag{
			Name:        "control-all",
			Aliases:     []string{"ctl-all"},
			Usage:       "enable container control",
			Destination: &conf.Control.All,
		},
		&cli.BoolFlag{
			Name:        "control-start",
			Aliases:     []string{"ctl-s"},
			Usage:       "enable container start  ",
			Destination: &conf.Control.Start,
		},
		&cli.BoolFlag{
			Name:        "control-stop",
			Aliases:     []string{"ctl-t"},
			Usage:       "enable container stop   ",
			Destination: &conf.Control.Stop,
		},
		&cli.BoolFlag{
			Name:        "control-restart",
			Aliases:     []string{"ctl-r"},
			Usage:       "enable container restart",
			Destination: &conf.Control.Restart,
		},
		&cli.BoolFlag{
			Name:    "help",
			Aliases: []string{"h"},
			Usage:   "show help",
		},
	}

	sort.Sort(cli.FlagsByName(appFlags))

	app := &cli.App{
		Name:      "container-web-tty",
		Usage:     "connect your containers via a web-tty",
		UsageText: "container-web-tty [global options]",
		Flags:     appFlags,
		HideHelp:  true,
		Authors:   author,
		Version: fmt.Sprintf("version: %s\tcommit: %s\tdate: %s",
			Version, CommitID, BuildAt),
		Action: func(c *cli.Context) error {
			if c.Bool("help") {
				return cli.ShowAppHelp(c)
			}
			// parse idleTime
			t := c.String("idle-time")
			idleTime, err := time.ParseDuration(t)
			if err != nil && t != "" {
				logrus.Fatalf("parse idle-time error: %s", err)
			} else {
				conf.Server.IdleTime = idleTime
			}

			if eArgs := c.String("extra-args"); eArgs != "" {
				conf.Backend.ExtraArgs = strings.Split(eArgs, " ")
			} else {
				switch conf.Backend.Type {
				case "docker":
					defaultArgs := "-e HISTCONTROL=ignoredups -e TERM=xterm"
					conf.Backend.ExtraArgs = strings.Split(defaultArgs, " ")
				case "kube":
				}
			}

			ctl := conf.Control
			if ctl.Start || ctl.Stop || ctl.Restart || ctl.All {
				conf.Control.Enable = true
			}

			conf.Server.Servers = strings.Split(c.String("servers"), " ")
			if conf.Debug {
				logrus.SetLevel(logrus.DebugLevel)
			} else {
				gin.SetMode(gin.ReleaseMode)
			}
			logrus.Debugf("got config: %+v", conf)

			run(c, *conf)
			return nil
		},
	}

	app.Run(os.Args)
}
