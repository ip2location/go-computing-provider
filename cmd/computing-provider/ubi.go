package main

import (
	_ "embed"
	"fmt"
	"github.com/filswan/go-mcs-sdk/mcs/api/common/logs"
	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	cors "github.com/itsjamie/gin-cors"
	"github.com/olekukonko/tablewriter"
	"github.com/swanchain/go-computing-provider/conf"
	"github.com/swanchain/go-computing-provider/internal/computing"
	"github.com/swanchain/go-computing-provider/internal/models"
	"github.com/swanchain/go-computing-provider/util"
	"github.com/urfave/cli/v2"
	"os"
	"strconv"
	"strings"
	"time"
)

var ubiTaskCmd = &cli.Command{
	Name:  "ubi",
	Usage: "Manage ubi tasks",
	Subcommands: []*cli.Command{
		detailCmd,
		listCmd,
		daemonCmd,
	},
}

var listCmd = &cli.Command{
	Name:  "list",
	Usage: "List ubi task",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name:  "show-failed",
			Usage: "show failed/failing ubi tasks",
		},
	},
	Action: func(cctx *cli.Context) error {
		cpRepoPath, _ := os.LookupEnv("CP_PATH")
		if err := conf.InitConfig(cpRepoPath, true); err != nil {
			return fmt.Errorf("load config file failed, error: %+v", err)
		}

		showFailed := cctx.Bool("show-failed")

		var taskData [][]string
		var rowColorList []RowColor
		var taskList []*models.TaskEntity
		var err error
		if showFailed {
			taskList, err = computing.NewTaskService().GetAllTask()
			if err != nil {
				return fmt.Errorf("failed get ubi task, error: %+v", err)
			}
		} else {
			taskList, err = computing.NewTaskService().GetTaskList(models.TASK_SUCCESS_STATUS)
			if err != nil {
				return fmt.Errorf("failed get ubi task, error: %+v", err)
			}
		}

		for i, task := range taskList {
			createTime := time.Unix(task.CreateTime, 0).Format("2006-01-02 15:04:05")
			taskData = append(taskData,
				[]string{strconv.Itoa(int(task.Id)), models.GetSourceTypeStr(task.ResourceType), task.ZkType, task.TxHash, models.TaskStatusStr(task.Status),
					fmt.Sprintf("%s", task.Reward), createTime})

			var rowColor []tablewriter.Colors
			if task.Status == models.TASK_RECEIVED_STATUS {
				rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgWhiteColor}}
			} else if task.Status == models.TASK_RUNNING_STATUS {
				rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgCyanColor}}
			} else if task.Status == models.TASK_SUCCESS_STATUS {
				rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgGreenColor}}
			} else if task.Status == models.TASK_FAILED_STATUS {
				rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgRedColor}}
			}

			rowColorList = append(rowColorList, RowColor{
				row:    i,
				column: []int{4},
				color:  rowColor,
			})
		}

		header := []string{"TASK ID", "TASK TYPE", "ZK TYPE", "PROOF HASH", "STATUS", "REWARD", "CREATE TIME"}
		NewVisualTable(header, taskData, rowColorList).Generate(true)

		return nil

	},
}

var detailCmd = &cli.Command{
	Name:      "detail",
	Usage:     "Use task_id to display task details",
	ArgsUsage: "[task_id]",
	Action: func(cctx *cli.Context) error {
		cpRepoPath, _ := os.LookupEnv("CP_PATH")
		if err := conf.InitConfig(cpRepoPath, true); err != nil {
			return fmt.Errorf("load config file failed, error: %+v", err)
		}

		taskIdStr := cctx.Args().Get(0)
		if strings.TrimSpace(taskIdStr) == "" {
			return fmt.Errorf("task_id is required")
		}
		taskId, err := strconv.ParseInt(taskIdStr, 10, 64)
		if err != nil {
			return err
		}
		taskEntity, err := computing.NewTaskService().GetTaskEntity(taskId)
		if err != nil {
			return fmt.Errorf("get %d task info failed, error: %v", taskId, err)
		}

		var taskData [][]string
		taskData = append(taskData, []string{"Task Name:", taskEntity.Name})
		taskData = append(taskData, []string{"ZK Type:", taskEntity.ZkType})
		taskData = append(taskData, []string{"Contract Address:", taskEntity.Contract})
		taskData = append(taskData, []string{"Task Status:", models.TaskStatusStr(taskEntity.Status)})
		taskData = append(taskData, []string{"Reward Status:", models.GetRewardStr(taskEntity.RewardStatus)})
		taskData = append(taskData, []string{"Reward:", taskEntity.Reward})
		taskData = append(taskData, []string{"Proof Tx Hash:", taskEntity.TxHash})
		if taskEntity.RewardTx != "" {
			taskData = append(taskData, []string{"Reward Tx Hash:", taskEntity.RewardTx})
		}
		if taskEntity.ChallengeTx != "" {
			taskData = append(taskData, []string{"Challenge Tx Hash:", taskEntity.ChallengeTx})
		}
		if taskEntity.SlashTx != "" {
			taskData = append(taskData, []string{"Slash Tx Hash:", taskEntity.SlashTx})
		}
		taskData = append(taskData, []string{"CreateTime:", time.Unix(taskEntity.CreateTime, 0).Format("2006-01-02 15:04:05")})

		var rowColorList []RowColor
		var rowColor []tablewriter.Colors
		if taskEntity.Status == models.TASK_RECEIVED_STATUS {
			rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgWhiteColor}}
		} else if taskEntity.Status == models.TASK_RUNNING_STATUS {
			rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgCyanColor}}
		} else if taskEntity.Status == models.TASK_SUCCESS_STATUS {
			rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgGreenColor}}
		} else if taskEntity.Status == models.TASK_FAILED_STATUS {
			rowColor = []tablewriter.Colors{{tablewriter.Bold, tablewriter.FgRedColor}}
		}
		rowColorList = append(rowColorList, RowColor{
			row:    3,
			column: []int{1},
			color:  rowColor,
		})

		header := []string{"Task Id:", taskIdStr}
		NewVisualTable(header, taskData, rowColorList).Generate(false)
		return nil

	},
}

var daemonCmd = &cli.Command{
	Name:  "daemon",
	Usage: "Start a cp process",

	Action: func(cctx *cli.Context) error {
		logs.GetLogger().Info("Start a computing-provider client.")
		cpRepoPath, _ := os.LookupEnv("CP_PATH")

		resourceExporterContainerName := "resource-exporter"
		rsExist, err := computing.NewDockerService().RemoveUnRunningContainer(resourceExporterContainerName)
		if err != nil {
			return fmt.Errorf("check %s container failed, error: %v", resourceExporterContainerName, err)
		}
		if !rsExist {
			if err = computing.RestartResourceExporter(); err != nil {
				logs.GetLogger().Errorf("restartResourceExporter failed, error: %v", err)
			}
		}
		if err := conf.InitConfig(cpRepoPath, true); err != nil {
			logs.GetLogger().Fatal(err)
		}

		computing.SyncCpAccountInfo()
		computing.CronTaskForEcp()

		r := gin.Default()
		r.Use(cors.Middleware(cors.Config{
			Origins:         "*",
			Methods:         "GET, PUT, POST, DELETE",
			RequestHeaders:  "Origin, Authorization, Content-Type",
			ExposedHeaders:  "",
			MaxAge:          50 * time.Second,
			ValidateHeaders: false,
		}))
		pprof.Register(r)

		v1 := r.Group("/api/v1")
		router := v1.Group("/computing")

		router.GET("/cp", computing.GetCpResource)
		router.POST("/cp/ubi", computing.DoUbiTaskForDocker)
		router.POST("/cp/docker/receive/ubi", computing.ReceiveUbiProofForDocker)

		shutdownChan := make(chan struct{})
		httpStopper, err := util.ServeHttp(r, "cp-api", ":"+strconv.Itoa(conf.GetConfig().API.Port), false)
		if err != nil {
			logs.GetLogger().Fatal("failed to start cp-api endpoint: %s", err)
		}

		finishCh := util.MonitorShutdown(shutdownChan,
			util.ShutdownHandler{Component: "cp-api", StopFunc: httpStopper},
		)
		<-finishCh

		return nil
	},
}
