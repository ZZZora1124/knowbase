package cmd

import (
	"context"
	"os"
	"os/signal"

	"github.com/spf13/cobra"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/cmd/brain"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/cmd/notebook"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/auth"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/config"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/loader"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/internal/storage"
	"icode.baidu.com/baidu/devops-ai/knowbase-client/pkg/types"
)

// Execute 连接已有子命令和配置加载流程；认证仅在具体业务执行时进行。
func Execute() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()

	var configPath string
	root := &cobra.Command{
		Use:           "knowbase",
		Short:         "知识方舟客户端",
		Args:          cobra.NoArgs,
		SilenceUsage:  true,
		SilenceErrors: true, // main 统一打印错误。
		RunE: func(cmd *cobra.Command, args []string) error {
			return runLoad(cmd.Context(), configPath)
		},
	}
	root.Flags().StringVarP(&configPath, "config", "c", "~/.knowledge/config.yaml", "知识配置文件路径")
	root.AddCommand(notebook.NotebookCmd, brain.BrainCmd)
	return root.ExecuteContext(ctx)
}

func runLoad(ctx context.Context, configPath string) error {
	environment := os.Getenv(types.EnvKnowbaseEnv)
	if environment == "" {
		environment = "prod"
	}
	if err := types.InitEnv(environment); err != nil {
		return err
	}
	parser, err := config.NewParser(configPath)
	if err != nil {
		return err
	}
	cfg, err := parser.Parse()
	if err != nil {
		return err
	}
	credential, err := auth.GetCredential()
	if err != nil {
		return err
	}
	localStorage, err := storage.NewLocalStorage(cfg.Storage.BasePath, cfg.PrimaryRepo)
	if err != nil {
		return err
	}
	return loader.NewManager(credential, localStorage, cfg).LoadAll(ctx)
}
