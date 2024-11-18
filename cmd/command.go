package cmd

import (
	"fmt"
	_ "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
	container "minidocker/container"
	cgroups "minidocker/container/cgroups"
)

var RunCommand = cli.Command{
	Name: "run",
	Usage: `Create a container with namespace and cgroups limit
			mydocker run -it [command]`,

	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it", // interactive terminal
			Usage: "enable print logs on tty, e.g.: -it",
		},
		cli.StringFlag{
			Name:  "mem",
			Usage: "memory max limit, e.g.: -mem 100m",
		},
		cli.StringFlag{
			Name:  "mem-min",
			Usage: "memory min limit, e.g.: -mem-min 100m",
		},
		cli.StringFlag{
			Name:  "mem-low",
			Usage: "memory low limit, e.g.: -mem-low 100m",
		},
		cli.StringFlag{
			Name:  "mem-high",
			Usage: "memory high limit, e.g.: -mem-high 100m",
		},
		cli.StringFlag{
			Name:  "mem-swap-max",
			Usage: "memory swap max limit, e.g.: -mem-swap-max 100m",
		},
		cli.StringFlag{
			Name: "cpu",
			Usage: "max cpu usage of this group, e.g.: -cpu-max 100",
		},
		cli.StringFlag{
			Name: "cpu-weight",
			Usage: "cpu weight of this group, e.g.: -cpu-weight 100",
		},
		cli.StringFlag{
			Name: "cpu-weight-nice",
			Usage: "cpu weight nice of this group, e.g.: -cpu-weight-nice 100",
		},
		cli.StringFlag{
			Name: "cpuset",
			Usage: "cpu set limit, e.g.: -cpuset 0-2 or -cpuset 0,1",
		},
	},
	Action: func(context *cli.Context) error {
		if len(context.Args()) < 1 {
			return fmt.Errorf("missing container command")
		}
		cmd := context.Args()
		tty := context.Bool("it")
		resConf := &cgroups.ResourceConfig{
			MemoryMax: context.String("mem"),
			MemoryMin: context.String("mem-min"),
			MemoryLow: context.String("mem-low"),
			MemoryHigh: context.String("mem-high"),
			MemorySwapMax: context.String("mem-swap-max"),
			CpuMax: context.String("cpu"),
			CpuWeight: context.String("cpu-weight"),
			CpuWeightNice: context.String("cpu-weight-nice"),
			CpuSet: context.String("cpuset"),
		}
		Run(cmd, tty, resConf)
		return nil
	},
}

var InitCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		err := container.InitContainerProcess()
		return err
	},
}
